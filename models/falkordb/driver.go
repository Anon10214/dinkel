/*
Package falkordb provides the model for FlakorDB, a Redis module.
*/
package falkordb

import (
	"fmt"
	"math"
	"reflect"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	rg "github.com/RedisGraph/redisgraph-go"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

// Driver for FlakorDB
type Driver struct {
	conn  redis.Conn
	graph rg.Graph
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
	d.conn, err = redis.Dial("tcp", fmt.Sprintf("%s:%d", opts.Host, port), redis.DialConnectTimeout(opts.Timeout))
	if err != nil {
		return err
	}

	d.graph = rg.GraphNew("", d.conn)

	return nil
}

// Reset the database
func (d *Driver) Reset(opts dbms.DBOptions) error {
	// Call flushall on the underlying redis instance
	_, err := redis.DoWithTimeout(d.conn, opts.Timeout, "FLUSHALL")
	return err
}

// RunQuery runs the query against the FlakorDB DB and returns its result.
func (d *Driver) RunQuery(opts dbms.DBOptions, query string) dbms.QueryResult {
	res := dbms.QueryResult{}
	// TODO: Don't ignore result
	returned, err := d.graph.ParameterizedQuery(query, map[string]any{"TIMEOUT_DEFAULT": fmt.Sprint(opts.Timeout.Seconds())})
	if err != nil {
		logrus.Debugf("Query produced error - %v", err)
		res.ProducedError = err
		return res
	}

	res.Schema = []any{}

	for returned.Next() {
		res.Rows = append(res.Rows, returned.Record().Values())
	}

	return res
}

// GetSchema returns the database's current schema
func (d *Driver) GetSchema(dbms.DBOptions) (*schema.Schema, error) {
	s := &schema.Schema{}
	s.Reset()
	return s, nil
}

// GetQueryResultType evaluates the produced result and returns the type the result indicates.
func (d *Driver) GetQueryResultType(res dbms.QueryResult, errorMessageRegex *dbms.ErrorMessageRegex) dbms.QueryResultType {
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
		logrus.Warnf("Encountered unknown error type %T: %v", err, err)
		return dbms.Bug
	}

	if errorMessageRegex.Ignored.MatchString(err.Error()) {
		return dbms.Invalid
	}

	if errorMessageRegex.Reported.MatchString(err.Error()) {
		return dbms.ReportedBug
	}

	logrus.Warnf("Encountered Redis Error: %v", err)
	return dbms.Bug
}

// DiscardQuery returns true with probability 1/3 or if the query produced a non-nil error,
// else it returns false. Thereby causing queries to have an expected amount of 4 statements
// if they don't produce an error.
func (d *Driver) DiscardQuery(res dbms.QueryResult, seed *seed.Seed) bool {
	if res.ProducedError != nil {
		return true
	}

	out := seed.GetByte()
	return out%3 == 0
}

// VerifyConnectivity checks whether the DB is still reachable and hasn't crashed.
func (d *Driver) VerifyConnectivity(opts dbms.DBOptions) (bool, error) {
	_, err := redis.DoWithTimeout(d.conn, opts.Timeout, "PING")
	if err != nil {
		// If connection closed, try to reconnect as it's using TCP
		err = d.Init(opts)
		if err != nil {
			return false, err
		}
		_, err := redis.DoWithTimeout(d.conn, opts.Timeout, "PING")
		return err == nil, err
	}
	return true, nil
}

// IsEqualResult returns whether the two passed results equal
func (d *Driver) IsEqualResult(a dbms.QueryResult, b dbms.QueryResult) bool {
	if len(a.Rows) != len(b.Rows) {
		logrus.Warn("Encountered mismatching results")
		logrus.Debugf("\n\t%v\nvs\n\t%v", a.Rows, b.Rows)
		return false
	}

	// Check that result rows match
	if !d.IsMatchingRows(a.Rows, b.Rows) {
		logrus.Warn("Encountered mismatching rows")
		logrus.Debugf("\n\t%v\nvs\n\t%v", a.Rows, b.Rows)
		return false
	}

	// Check if the schemas match
	if !d.IsMatchingRows(a.Schema, b.Schema) {
		logrus.Warnf("Mismatching Schemas")
		logrus.Debugf("Schemas:\n\t%+v\nvs\n\t%+v", a.Schema, b.Schema)
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
	firstRow := first.([]any)
	secondRow := second.([]any)
	// If only one row is nil
	if (firstRow == nil || secondRow == nil) && (firstRow != nil || secondRow != nil) {
		return false
	}
	if len(firstRow) != len(secondRow) {
		return false
	}

	for i := range firstRow {
		matchedIndex := -1
		for j := range secondRow {
			if isMatchingRedisGraphElement(firstRow[i], secondRow[j]) {
				matchedIndex = j
				break
			}
		}
		if matchedIndex == -1 {
			return false
		}
		secondRow = append(secondRow[:matchedIndex], secondRow[matchedIndex+1:]...)
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
	case *rg.Node:
		b, ok := secondElement.(*rg.Node)
		return ok && (isMatchingNode(*a, *b))
	case *rg.Edge:
		b, ok := secondElement.(*rg.Edge)
		return ok && (isMatchingEdge(*a, *b))
	default:
		fmt.Printf("%T\n", a)
		if !reflect.DeepEqual(a, b) {
			return false
		}
	}
	return true
}

func isMatchingNode(a, b rg.Node) bool {
	if len(a.Labels) != len(b.Labels) {
		return false
	}
	for i := range a.Labels {
		if a.Labels[i] != b.Labels[i] {
			return false
		}
	}
	return reflect.DeepEqual(a.Properties, b.Properties)
}

func isMatchingEdge(a, b rg.Edge) bool {
	if a.Relation != b.Relation {
		return false
	}
	return reflect.DeepEqual(a.Properties, b.Properties)
}
