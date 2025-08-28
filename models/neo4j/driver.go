/*
Package neo4j provides the model for Neo4j
*/
package neo4j

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sirupsen/logrus"
)

// Driver for neo4j
type Driver struct {
	driver  neo4j.DriverWithContext
	session neo4j.SessionWithContext

	opts dbms.DBOptions
}

// Init the DB driver
func (d *Driver) Init(opts dbms.DBOptions) error {
	d.opts = opts
	connPort := 7687
	if opts.Port != nil {
		connPort = *opts.Port
	}
	driver, err := neo4j.NewDriverWithContext(fmt.Sprintf("bolt://%s:%d", opts.Host, connPort), neo4j.NoAuth(), func(c *neo4j.Config) {
		c.ConnectionAcquisitionTimeout = opts.Timeout
	})
	if err != nil {
		return err
	}
	d.driver = driver
	d.opts.BackwardsCompatibleMode = true

	logrus.Debug("Setting up connection to the neo4j database")
	return nil
}

// Reset the database
func (d *Driver) Reset(opts dbms.DBOptions) error {
	logrus.Debug("Resetting Database")
	ctx := context.Background()
	d.session = d.driver.NewSession(context.Background(), neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	if _, err := d.session.ExecuteWrite(ctx, func(transaction neo4j.ManagedTransaction) (any, error) {
		// Delete nodes and relationships
		if _, err := transaction.Run(ctx, "MATCH (n) DETACH DELETE n", nil); err != nil {
			return nil, errors.Join(fmt.Errorf("error while deleting nodes and relationships %v", err))
		}
		return nil, nil
	}, neo4j.WithTxTimeout(opts.Timeout)); err != nil {
		logrus.Errorf("couldn't reset database - %v", err)
	}

	// Don't call the apoc procedure in backwards compatible mode
	if d.opts.BackwardsCompatibleMode {
		return nil
	}

	if _, err := d.session.ExecuteWrite(ctx, func(transaction neo4j.ManagedTransaction) (any, error) {
		// Reset indexes and constraints
		if _, err := transaction.Run(ctx, "CALL apoc.schema.assert({},{},true) YIELD label, key RETURN *", nil); err != nil {
			return nil, errors.Join(fmt.Errorf("error while resetting indexes and constraints: %v", err))
		}
		return nil, nil
	}, neo4j.WithTxTimeout(opts.Timeout)); err != nil {
		logrus.Errorf("couldn't reset database - %v", err)
	}

	if _, err := d.session.ExecuteWrite(ctx, func(transaction neo4j.ManagedTransaction) (any, error) {
		// Clear query cache
		if _, err := transaction.Run(ctx, "CALL db.clearQueryCaches", nil); err != nil {
			return nil, errors.Join(fmt.Errorf("error while clearing query cache: %v", err))
		}
		return nil, nil
	}, neo4j.WithTxTimeout(opts.Timeout)); err != nil {
		logrus.Errorf("couldn't clear query cache - %v", err)
	}

	return nil
}

// RunQuery runs the query against the Neo4j DB and returns its result.
func (d Driver) RunQuery(opts dbms.DBOptions, query string) dbms.QueryResult {
	ctx := context.Background()
	logrus.Debug("Sending query to database")
	var queryResult dbms.QueryResult
	// Ignore the result, only consider err, (maybe use it later for statistics?)
	if _, err := d.session.ExecuteWrite(ctx, func(transaction neo4j.ManagedTransaction) (any, error) {
		// Get the result for the query to execute
		res, err := transaction.Run(ctx, query, nil)
		queryResult = dbms.QueryResult{
			ProducedError: err,
		}
		if err != nil {
			return nil, err
		}
		for res.Next(ctx) {
			queryResult.Rows = append(queryResult.Rows, res.Record().Values)
		}

		queryResult.ProducedError = res.Err()
		return nil, res.Err()
	}, neo4j.WithTxTimeout(opts.Timeout)); err != nil {
		logrus.Debugf("Error %v produced when running query %s", err, query)
		return queryResult
	}

	if _, err := d.session.ExecuteRead(ctx, func(transaction neo4j.ManagedTransaction) (any, error) {
		queryResult.Schema = []any{}

		res, err := transaction.Run(ctx, "MATCH (n) RETURN n AS x UNION MATCH ()-[m]-() RETURN m AS x", nil)
		if err != nil {
			queryResult.ProducedError = err
			return nil, err
		}

		for res.Next(ctx) {
			queryResult.Schema = append(queryResult.Schema.([]any), res.Record().Values)
		}

		if res.Err() != nil {
			queryResult.ProducedError = res.Err()
		}
		return nil, res.Err()
	}, neo4j.WithTxTimeout(opts.Timeout)); err != nil {
		logrus.Debugf("Error %v produced when trying to get schema", err)
		return queryResult
	}

	logrus.Debug("Query finished")
	return queryResult
}

// GetSchema returns the database's current schema
func (d Driver) GetSchema(opts dbms.DBOptions) (*schema.Schema, error) {
	s := &schema.Schema{}
	s.Reset()

	if err := d.populateLabels(opts, s); err != nil {
		logrus.Errorf("Error while populating labels of schema: %v", err)
		return nil, err
	}
	logrus.Tracef("Populated schema with labels %v", s.Labels)
	if err := d.populateProperties(opts, s); err != nil {
		logrus.Errorf("Error while populating properties of schema: %v", err)
		return nil, err
	}
	logrus.Tracef("Populated schema with properties %v", s.Properties)

	return s, nil
}

// populateLabels fetches all labels in the DB and inserts them into the schema.
func (d Driver) populateLabels(opts dbms.DBOptions, s *schema.Schema) error {
	ctx := context.Background()
	_, err := d.session.ExecuteWrite(ctx, func(transaction neo4j.ManagedTransaction) (any, error) {
		// Get node labels
		res, err := transaction.Run(ctx, "MATCH (n) UNWIND labels(n) AS label RETURN DISTINCT label ORDER BY label", nil)
		if err != nil {
			return nil, err
		}
		for res.Next(ctx) {
			s.Labels[schema.NODE] = append(s.Labels[schema.NODE], res.Record().Values[0].(string))
		}

		// Get relationship labels (also called types)
		res, err = transaction.Run(ctx, "MATCH ()-[m]-() RETURN DISTINCT type(m) AS label ORDER BY label", nil)
		if err != nil {
			return nil, err
		}
		for res.Next(ctx) {
			s.Labels[schema.RELATIONSHIP] = append(s.Labels[schema.RELATIONSHIP], res.Record().Values[0].(string))
		}

		s.Labels[schema.ANY] = append(s.Labels[schema.RELATIONSHIP], s.Labels[schema.NODE]...)

		return nil, nil
	}, neo4j.WithTxTimeout(opts.Timeout))
	return err
}

// propertyStringToType takes in a string of a type as returned
// by the neo4j DB and returns its equivalent [schema.PropertyType].
func propertyStringToType(neo4jType any) schema.PropertyType {
	mask := 0

	varType, found := strings.CutPrefix(neo4jType.(string), "LIST OF ")
	if found {
		mask = schema.ListMask
	}

	propType, found := map[string]schema.PropertyType{
		"BOOLEAN":         schema.Boolean,
		"DATE":            schema.Date,
		"DATE_TIME":       schema.Datetime,
		"DURATION":        schema.Duration,
		"FLOAT":           schema.Float,
		"INTEGER":         schema.Integer,
		"LOCAL_DATE_TIME": schema.LocalDateTime,
		"LOCAL_TIME":      schema.LocalTime,
		"POINT":           schema.Point,
		"STRING":          schema.String,
		"TIME":            schema.Time,
	}[varType]
	if !found {
		logrus.Errorf("Couldn't identify type %s while populating properties", varType)
	}

	return schema.PropertyType(mask) | propType
}

// populateProperties fetches all properties in the DB and inserts them into the schema.
func (d Driver) populateProperties(opts dbms.DBOptions, s *schema.Schema) error {
	ctx := context.Background()
	_, err := d.session.ExecuteWrite(ctx, func(transaction neo4j.ManagedTransaction) (any, error) {
		res, err := transaction.Run(ctx, `
		MATCH ()-[m]-() UNWIND keys(properties(m)) AS propKey RETURN DISTINCT propKey AS key, apoc.meta.cypher.type(m[propKey]) AS type, m[propKey] AS value ORDER BY key, type, value
			UNION
		MATCH (n) UNWIND keys(properties(n)) AS propKey RETURN DISTINCT propKey AS key, apoc.meta.cypher.type(n[propKey]) AS type, n[propKey] AS value ORDER BY key, type, value
		`, nil)
		if err != nil {
			return nil, err
		}

		for res.Next(ctx) {
			property := schema.Property{
				Name: res.Record().Values[0].(string),
				Type: propertyStringToType(res.Record().Values[1]),
			}
			switch val := res.Record().Values[2].(type) {
			case neo4j.Point2D:
				property.Value = fmt.Sprintf("point({x: %f, y: %f})", val.X, val.Y)
			case neo4j.Point3D:
				property.Value = fmt.Sprintf("point({x: %f, y: %f, z: %f})", val.X, val.Y, val.Z)
			case string:
				property.Value = fmt.Sprintf(`"%s"`, val)
			// TODO: Implement all the rest of the data types
			case neo4j.Date:
				property.Value = "date('2000-01-01')"
			case neo4j.LocalDateTime:
				property.Value = "localdatetime('2015185T19:32:24')"
			case neo4j.LocalTime:
				property.Value = "localtime('12:50:35.556')"
			case neo4j.Time:
				property.Value = "time('125035.556+0100')"
			case neo4j.Duration:
				property.Value = `duration("P1Y")`
			case time.Time:
				property.Value = "datetime('2015-06-24T12:50:35.556+0100')"
			default:
				property.Value = fmt.Sprint(res.Record().Values[2])
			}
			s.AddProperty(property)
		}
		return nil, nil
	}, neo4j.WithTxTimeout(opts.Timeout))
	return err
}

// IsMatchingRows Compares the string representations of two neo4j result rows.
// It returns true if they match, else false.
func IsMatchingRows(first, second any) bool {
	if first == nil || second == nil {
		// If only one row is nil
		if first != nil || second != nil {
			return false
		}
		// If both are nil
		return true
	}

	firstRow := first.([]any)
	secondRow := second.([]any)
	// Copy second row since it gets manipulated during the comparison
	secondRowCopy := make([]any, len(secondRow))
	copy(secondRowCopy, secondRow)

	if len(firstRow) != len(secondRow) {
		return false
	}

	for i := range firstRow {
		matchedIndex := -1
		for j := range secondRowCopy {
			if isMatchingNeo4jElement(firstRow[i], secondRowCopy[j]) {
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

// isMatchingNeo4jElement returns true if the two passed elements
// evaluate to the same neo4j elements, else it returns false.
func isMatchingNeo4jElement(a, b any) bool {
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
				if !isMatchingNeo4jElement(a[i], b[i]) {
					return false
				}
			}
			return true
		}
		return false
	case neo4j.Node:
		b, ok := secondElement.(neo4j.Node)
		if !(ok && isMatchingNode(a, b)) {
			return false
		}
	case neo4j.Relationship:
		b, ok := secondElement.(neo4j.Relationship)
		if !(ok && isMatchingRelationship(a, b)) {
			return false
		}
	case neo4j.Path:
		b, ok := secondElement.(neo4j.Path)
		if !ok {
			return false
		}
		if len(a.Nodes) != len(b.Nodes) || len(a.Relationships) != len(b.Relationships) {
			return false
		}
		for i := range a.Nodes {
			if !isMatchingNode(a.Nodes[i], b.Nodes[i]) {
				return false
			}
		}
		for i := range a.Relationships {
			if !isMatchingRelationship(a.Relationships[i], b.Relationships[i]) {
				return false
			}
		}
	case float64:
		b, ok := secondElement.(float64)
		return ok && (b == a || (math.IsNaN(a) && math.IsNaN(b)))
	default:
		if !reflect.DeepEqual(a, b) {
			return false
		}
	}
	return true
}

func isMatchingNode(a, b neo4j.Node) bool {
	if len(a.Labels) != len(b.Labels) {
		return false
	}
	for i := range a.Labels {
		if a.Labels[i] != b.Labels[i] {
			return false
		}
	}

	if len(a.Props) != len(b.Props) {
		return false
	}

	for k, v := range a.Props {
		if !isMatchingNeo4jElement(v, b.Props[k]) {
			return false
		}
	}
	return true
}

func isMatchingRelationship(a, b neo4j.Relationship) bool {
	if a.Type != b.Type {
		return false
	}

	if len(a.Props) != len(b.Props) {
		return false
	}

	for k, v := range a.Props {
		if !isMatchingNeo4jElement(v, b.Props[k]) {
			return false
		}
	}
	return true
}

// GetQueryResultType evaluates the produced result and returns the type the result indicates.
func (d Driver) GetQueryResultType(res dbms.QueryResult, errorMessageRegex *dbms.ErrorMessageRegex) dbms.QueryResultType {
	err := res.ProducedError
	if err == nil {
		return dbms.Valid
	}

	switch err := err.(type) {
	case *neo4j.ConnectivityError:
		return dbms.Timeout
	case *neo4j.Neo4jError:
		// A user-defined timeout
		if err.Title() == "TransactionTimedOutClientConfiguration" {
			return dbms.Timeout
		}

		// TODO: Move below comment to docs somewhere
		// Don't count these as bugs
		// Generation itself should try its best to avoid syntax and semantic errors
		// Error messages here should be unavoidable or resource-heavy to detect during generation

		if errorMessageRegex.Ignored.MatchString(err.Msg) {
			return dbms.Invalid
		}

		// TODO: Move below comment to docs somewhere
		// These error messages indicate bugs which have already been reported

		if errorMessageRegex.Reported.MatchString(err.Msg) {
			return dbms.ReportedBug
		}

		logrus.Warnf("Encountered Neo4jError with error title %q and msg %s", err.Title(), err.Msg)
		return dbms.Bug
	default:
		return dbms.Bug
	}
}

// DiscardQuery returns true with probability 1/10 or if the query produced a non-nil error,
// else it returns false. Thereby causing queries to have an expected amount of 11 statements
// if they don't produce an error.
func (d Driver) DiscardQuery(res dbms.QueryResult, seed *seed.Seed) bool {
	if res.ProducedError != nil {
		return true
	}

	return seed.BooleanWithProbability(0.1)
}

// VerifyConnectivity checks whether the DB is still reachable and hasn't crashed.
func (d Driver) VerifyConnectivity(dbms.DBOptions) (bool, error) {
	err := d.driver.VerifyConnectivity(context.Background())
	return err == nil, err
}

// IsEqualResult returns true if the two passed query results hold the same information, else false.
func (d Driver) IsEqualResult(a, b dbms.QueryResult) bool {
	if len(a.Rows) != len(b.Rows) {
		logrus.Warn("Encountered mismatching results")
		logrus.Infof("\n\t%v\nvs\n\t%v", a.Rows, b.Rows)
		return false
	}

	// Check that result rows match
	if !IsMatchingRows(a.Rows, b.Rows) {
		logrus.Warn("Encountered mismatching rows")
		logrus.Infof("\n\t%v\nvs\n\t%v", a.Rows, b.Rows)
		return false
	}

	// Check if the schemas match
	if !IsMatchingRows(a.Schema, b.Schema) {
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
