/*
Package memgraph provides the model for memgraph
*/
package memgraph

import (
	"context"
	"fmt"
	"strings"

	"github.com/Anon10214/dinkel/dbms"
	neo4jimpl "github.com/Anon10214/dinkel/models/neo4j"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sirupsen/logrus"
)

// Driver for memgraph
type Driver struct {
	driver  neo4j.DriverWithContext
	session neo4j.SessionWithContext
}

// Init the DB driver
func (d *Driver) Init(opts dbms.DBOptions) error {
	connPort := 7687
	if opts.Port != nil {
		connPort = *opts.Port
	}
	driver, err := neo4j.NewDriverWithContext(fmt.Sprintf("bolt://%s:%d", opts.Host, connPort), neo4j.NoAuth(), func(c *neo4j.Config) {
		c.ConnectionAcquisitionTimeout = opts.Timeout
		c.MaxTransactionRetryTime = 0
	})
	if err != nil {
		return err
	}
	d.driver = driver

	// Set the query timeout
	if _, err := driver.NewSession(context.Background(), neo4j.SessionConfig{}).Run(context.Background(), fmt.Sprintf(`SET DATABASE SETTING "query.timeout" TO "%f";`, opts.Timeout.Seconds()), nil, neo4j.WithTxTimeout(opts.Timeout)); err != nil {
		logrus.Errorf("couldn't set database timeout - %v", err)
		return err
	}

	logrus.Debug("Setting up connection to the memgraph database")
	return nil
}

// Reset the database
func (d *Driver) Reset(opts dbms.DBOptions) error {
	logrus.Debug("Resetting Database")
	ctx := context.Background()
	d.session = d.driver.NewSession(context.Background(), neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})

	// Delete all nodes and edges
	if _, err := d.session.Run(ctx, "MATCH (n) DETACH DELETE n", nil, neo4j.WithTxTimeout(opts.Timeout)); err != nil {
		logrus.Errorf("couldn't reset database - %v", err)
		return err
	}

	// Delete all constraints and indexes

	return nil
}

// RunQuery runs the query against the memgraph DB and returns its result.
func (d Driver) RunQuery(opts dbms.DBOptions, query string) dbms.QueryResult {
	ctx := context.Background()
	logrus.Debug("Sending query to database")

	var queryResult dbms.QueryResult
	var res neo4j.ResultWithContext
	var err error

	// Run the query
	if res, err = d.session.Run(ctx, query, nil, neo4j.WithTxTimeout(opts.Timeout)); err != nil {
		queryResult = dbms.QueryResult{
			ProducedError: err,
		}
		logrus.Debugf("Error %v produced when running query %s", err, query)
		return queryResult
	}
	for res.Next(ctx) {
		queryResult.Rows = append(queryResult.Rows, res.Record().Values)
	}

	queryResult.ProducedError = res.Err()
	if res.Err() != nil {
		logrus.Debugf("Error %v produced when running query %s", err, query)
		return queryResult
	}

	// Get query result schema
	queryResult.Schema = []any{}
	if res, err = d.session.Run(ctx, "MATCH (n) RETURN n AS x UNION MATCH ()-[m]-() RETURN m AS x", nil, neo4j.WithTxTimeout(opts.Timeout)); err != nil {
		queryResult.ProducedError = err
		logrus.Debugf("Error %v produced when trying to get schema", err)
		return queryResult
	}
	for res.Next(ctx) {
		queryResult.Schema = append(queryResult.Schema.([]any), res.Record().Values)
	}
	if res.Err() != nil {
		logrus.Debugf("Error %v produced when trying to get schema", err)
	}

	logrus.Debug("Query finished")
	return queryResult
}

// GetSchema returns the database's current schema
func (d Driver) GetSchema(opts dbms.DBOptions) (*schema.Schema, error) {
	s := &schema.Schema{}
	s.Reset()

	ctx := context.Background()

	nodes, err := d.session.Run(ctx, "MATCH (n) UNWIND labels(n) AS i RETURN DISTINCT i", nil, neo4j.WithTxTimeout(opts.Timeout))
	if err != nil {
		logrus.Errorf("Couldn't get nodes for schema - %v", err)
		return nil, err
	}
	for nodes.Next(ctx) {
		label := nodes.Record().Values[0].(string)
		s.Labels[schema.NODE] = append(s.Labels[schema.NODE], label)
	}

	relationships, err := d.session.Run(ctx, "MATCH ()-[n]-() RETURN DISTINCT type(n)", nil, neo4j.WithTxTimeout(opts.Timeout))
	if err != nil {
		logrus.Errorf("Couldn't get relationships for schema - %v", err)
		return nil, err
	}
	for relationships.Next(ctx) {
		label := relationships.Record().Values[0].(string)
		s.Labels[schema.RELATIONSHIP] = append(s.Labels[schema.RELATIONSHIP], label)
	}

	s.Labels[schema.ANY] = append(s.Labels[schema.RELATIONSHIP], s.Labels[schema.NODE]...)

	return s, nil
}

// GetQueryResultType evaluates the produced result and returns the type the result indicates.
func (d Driver) GetQueryResultType(res dbms.QueryResult, errorMessageRegex *dbms.ErrorMessageRegex) dbms.QueryResultType {
	err := res.ProducedError
	if err == nil {
		return dbms.Valid
	}

	switch err := err.(type) {
	case *neo4j.ConnectivityError:
		return dbms.Crash
	case *neo4j.Neo4jError:
		if err.Msg == "Transaction was asked to abort because of transaction timeout." {
			return dbms.Timeout
		}

		if err.Title() == "MemgraphError" {
			if errorMessageRegex.Ignored.MatchString(err.Msg) {
				return dbms.Invalid
			}

			if errorMessageRegex.Reported.MatchString(err.Msg) {
				return dbms.ReportedBug
			}
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
func (d Driver) VerifyConnectivity(opts dbms.DBOptions) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()
	err := d.driver.VerifyConnectivity(ctx)
	if err != nil && strings.HasPrefix(err.Error(), "error could not acquire server lock in time when cleaning up pool") {
		return true, nil
	}
	return err == nil, err
}

// IsEqualResult panics for memgraph, as it hasn't been implemented yet
func (d Driver) IsEqualResult(a dbms.QueryResult, b dbms.QueryResult) bool {
	if len(a.Rows) != len(b.Rows) {
		logrus.Warn("Encountered mismatching results")
		logrus.Infof("\n\t%v\nvs\n\t%v", a.Rows, b.Rows)
		return false
	}

	// Check that result rows match
	if !neo4jimpl.IsMatchingRows(a.Rows, b.Rows) {
		logrus.Warn("Encountered mismatching rows")
		logrus.Infof("\n\t%v\nvs\n\t%v", a.Rows, b.Rows)
		return false
	}

	// Check if the schemas match
	if !neo4jimpl.IsMatchingRows(a.Schema, b.Schema) {
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
