/*
Package falkordb provides the model for FlakorDB, a Redis module.
*/
package falkordb

import (
	"context"
	"fmt"
	"math"
	"net"
	"reflect"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/FalkorDB/falkordb-go"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Driver for FalkorDB
type Driver struct {
	conn    *redis.Client
	fdbConn *falkordb.FalkorDB
	graph   *falkordb.Graph

	// Due to a bug in the FalkorDB driver, it sometimes returns nil record values that falsely indicate a logic bug.
	// If this happens, treat the query as invalid.
	returnedNil bool

	// The amount of queries run, due to redis not killing long running processes if they write data, we just shutdown the server every once in a while
	ranQueries int
}

// Init the DB driver
func (d *Driver) Init(opts dbms.DBOptions) error {
	var port int
	if opts.Port == nil {
		port = 6379
	} else {
		port = *opts.Port
	}

	var err error
	d.fdbConn, err = falkordb.FalkorDBNew(&falkordb.ConnectionOption{
		Addr:         fmt.Sprintf("%s:%d", opts.Host, port),
		DialTimeout:  opts.Timeout,
		ReadTimeout:  opts.Timeout,
		WriteTimeout: opts.Timeout,
		PoolTimeout:  opts.Timeout,
		Protocol:     3,
		MaxRetries:   -1,
	})
	if err != nil {
		return err
	}

	d.graph = d.fdbConn.SelectGraph("")
	d.conn = d.graph.Conn

	return d.conn.Ping(context.Background()).Err()
}

// Reset the database
func (d *Driver) Reset(opts dbms.DBOptions) error {
	if d.ranQueries >= 10 {
		logrus.Debugf("Ran %d queries, shutting down redis to restart", d.ranQueries)
		d.ranQueries = 0
		d.conn.ShutdownNoSave(context.Background())
		for i := 0; ; i++ {
			if err := d.Init(opts); err == nil {
				break
			}
			if i == 100 {
				return fmt.Errorf("failed to reestablish connection to redis after shutting it down within %d tries - make sure it restarts on exit", i)
			}
		}
	}
	d.graph = d.fdbConn.SelectGraph("")
	d.conn = d.graph.Conn

	d.returnedNil = false
	// Call flushall on the underlying redis instance
	return d.conn.FlushAll(context.Background()).Err()
}

// RunQuery runs the query against the FlakorDB DB and returns its result.
func (d *Driver) RunQuery(opts dbms.DBOptions, query string) dbms.QueryResult {
	res := dbms.QueryResult{}

	queryOpts := falkordb.NewQueryOptions().SetTimeout(int(opts.Timeout.Seconds()))
	// Older versions only allow timeouts on read queries
	if opts.BackwardsCompatibleMode {
		queryOpts = nil
	}

	returned, err := d.graph.Query(query, nil, queryOpts)
	if err != nil {
		logrus.Debugf("Query produced error - %v", err)
		res.ProducedError = err
		return res
	}

	for returned.Next() {
		val := returned.Record()
		if val != nil {
			res.Rows = append(res.Rows, val.Values())
		} else {
			d.returnedNil = true
			logrus.Debugf("nil record after query, this is a bug with the FalkorDB driver %s", query)
		}
	}

	res.Schema = []any{}
	schemaRes, err := d.graph.Query("MATCH (n) RETURN n AS x UNION MATCH ()-[m]-() RETURN m AS x", nil, falkordb.NewQueryOptions().SetTimeout(int(opts.Timeout.Seconds())))
	if err != nil {
		logrus.Debugf("Couldn't get schema - %v", err)
		res.ProducedError = err
		return res
	}
	for schemaRes.Next() {
		val := schemaRes.Record()
		if val != nil {
			res.Schema = append(res.Schema.([]any), val.Values())
		} else {
			d.returnedNil = true
			logrus.Debugf("nil record after fetching schema after query, this is a bug with the FalkorDB driver %s", query)
		}
	}

	return res
}

// GetSchema returns the database's current schema
func (d *Driver) GetSchema(opts dbms.DBOptions) (*schema.Schema, error) {
	s := &schema.Schema{}
	s.Reset()
	nodes, err := d.graph.Query("MATCH (n) UNWIND labels(n) AS i RETURN DISTINCT i", nil, falkordb.NewQueryOptions().SetTimeout(int(opts.Timeout.Seconds())))
	if err != nil {
		logrus.Errorf("Couldn't get nodes for schema - %v", err)
		return nil, err
	}
	for nodes.Next() {
		label := nodes.Record().Values()[0].(string)
		s.Labels[schema.NODE] = append(s.Labels[schema.NODE], label)
	}

	relationships, err := d.graph.Query("MATCH ()-[n]-() RETURN DISTINCT type(n)", nil, falkordb.NewQueryOptions().SetTimeout(int(opts.Timeout.Seconds())))
	if err != nil {
		logrus.Errorf("Couldn't get relationships for schema - %v", err)
		return nil, err
	}
	for relationships.Next() {
		label := relationships.Record().Values()[0].(string)
		s.Labels[schema.RELATIONSHIP] = append(s.Labels[schema.RELATIONSHIP], label)
	}

	s.Labels[schema.ANY] = append(s.Labels[schema.RELATIONSHIP], s.Labels[schema.NODE]...)

	return s, nil
}

// GetQueryResultType evaluates the produced result and returns the type the result indicates.
func (d *Driver) GetQueryResultType(res dbms.QueryResult, errorMessageRegex *dbms.ErrorMessageRegex) dbms.QueryResultType {
	if d.returnedNil {
		return dbms.Invalid
	}

	err := res.ProducedError
	if err == nil {
		return dbms.Valid
	}

	// Probably indicates a crash
	// But let scheduler check if DB down
	if err.Error() == "EOF" {
		return dbms.Valid
	}

	if _, ok := err.(redis.Error); !ok {
		if err, ok := err.(*net.OpError); ok {
			logrus.Debugf("Encountered net.OpError %v", err)
			// TODO: Change this once fixed
			return dbms.ReportedBug
		}
		logrus.Warnf("Encountered unknown error type %T: %v", err, err)
		return dbms.Bug
	}

	if errorMessageRegex.Ignored.MatchString(err.Error()) {
		return dbms.Invalid
	}

	if errorMessageRegex.Reported.MatchString(err.Error()) {
		return dbms.ReportedBug
	}

	if err.Error() == "Query timed out" {
		return dbms.Timeout
	}

	logrus.Warnf("Encountered Redis Error: %v", err)
	return dbms.Bug
}

// DiscardQuery returns true with probability 1/10 or if the query produced a non-nil error,
// else it returns false. Thereby causing queries to have an expected amount of 11 statements
// if they don't produce an error.
func (d *Driver) DiscardQuery(res dbms.QueryResult, seed *seed.Seed) bool {
	if res.ProducedError != nil {
		d.ranQueries++
		return true
	}

	discard := seed.BooleanWithProbability(0.1)
	if discard {
		d.ranQueries++
	}
	return discard
}

// VerifyConnectivity checks whether the DB is still reachable and hasn't crashed.
func (d *Driver) VerifyConnectivity(opts dbms.DBOptions) (bool, error) {
	err := d.conn.Ping(context.Background()).Err()
	if err != nil {
		return false, err
	}
	return true, nil
}

// IsEqualResult returns whether the two passed results equal
func (d *Driver) IsEqualResult(a dbms.QueryResult, b dbms.QueryResult) bool {
	if len(a.Rows) != len(b.Rows) {
		logrus.Warn("Encountered mismatching results")
		logrus.Infof("\n\t%v\nvs\n\t%v", a.Rows, b.Rows)
		return false
	}

	// Check that result rows match
	if !d.IsMatchingRows(a.Rows, b.Rows) {
		logrus.Warn("Encountered mismatching rows")
		logrus.Infof("\n\t%v\nvs\n\t%v", a.Rows, b.Rows)
		return false
	}

	// Check if the schemas match
	if !d.IsMatchingRows(a.Schema, b.Schema) {
		logrus.Warnf("Mismatching Schemas")
		logrus.Infof("Schemas:\n\t%+v\nvs\n\t%+v", a.Schema, b.Schema)
		return false
	}

	if a.ProducedError != nil || b.ProducedError != nil {
		if a.ProducedError == nil || b.ProducedError == nil {
			return false
		}
		if a.ProducedError.Error() != b.ProducedError.Error() {
			return false
		}
	}

	return true
}

// IsMatchingRows compares two redisgraph result rows.
// It returns true if they match, else false.
func (d *Driver) IsMatchingRows(first, second any) bool {
	if first == nil || second == nil {
		return first == nil && second == nil
	}

	firstRow := first.([]any)
	secondRow := second.([]any)
	// Copy second row since it gets manipulated during the comparison
	secondRowCopy := make([]any, len(secondRow))
	copy(secondRowCopy, secondRow)

	// If only one row is nil
	if (firstRow == nil || secondRow == nil) && (firstRow != nil || secondRow != nil) {
		return false
	}
	if len(firstRow) != len(secondRow) {
		return false
	}

	for i := range firstRow {
		matchedIndex := -1
		for j := range secondRowCopy {
			if isMatchingRedisGraphElement(firstRow[i], secondRowCopy[j]) {
				matchedIndex = j
				break
			}
		}
		if matchedIndex == -1 {
			return false
		}
		secondRowCopy = append(secondRowCopy[:matchedIndex], secondRowCopy[matchedIndex+1:]...)
	}
	return true
}

// isMatchingRedisGraphElement returns true if the two passed elements
// evaluate to the same redisgraph elements, else it returns false.
func isMatchingRedisGraphElement(a, b any) bool {
	secondElement := b
	switch a := a.(type) {
	case nil:
		return b == nil
	case []any:
		if b, ok := secondElement.([]any); ok {
			if len(a) != len(b) {
				return false
			}
			for i := range a {
				if !isMatchingRedisGraphElement(a[i], b[i]) {
					return false
				}
			}
			return true
		}
		return false
	case int:
		b, ok := secondElement.(int)
		return ok && (b == a)
	case float64:
		b, ok := secondElement.(float64)
		return ok && (b == a || (math.IsNaN(a) && math.IsNaN(b)))
	case *falkordb.Node:
		b, ok := secondElement.(*falkordb.Node)
		return ok && (isMatchingNode(*a, *b))
	case *falkordb.Edge:
		b, ok := secondElement.(*falkordb.Edge)
		return ok && (isMatchingEdge(*a, *b))
	default:
		if !reflect.DeepEqual(a, b) {
			return false
		}
	}
	return true
}

func isMatchingNode(a, b falkordb.Node) bool {
	if len(a.Labels) != len(b.Labels) {
		return false
	}
	// Mismatching labels
	if cmp.Diff(a.Labels, b.Labels, cmpopts.SortSlices(func(a, b string) bool { return a < b })) != "" {
		return false
	}

	if len(a.Properties) != len(b.Properties) {
		return false
	}

	for k, v := range a.Properties {
		if !isMatchingRedisGraphElement(v, b.Properties[k]) {
			return false
		}
	}
	return true
}

func isMatchingEdge(a, b falkordb.Edge) bool {
	if a.Relation != b.Relation {
		return false
	}

	if len(a.Properties) != len(b.Properties) {
		return false
	}

	for k, v := range a.Properties {
		if !isMatchingRedisGraphElement(v, b.Properties[k]) {
			return false
		}
	}

	return true
}
