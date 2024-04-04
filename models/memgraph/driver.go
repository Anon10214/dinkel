/*
Package memgraph provides the model for memgraph
*/
package memgraph

import (
	"context"
	"errors"
	"fmt"

	"github.com/Anon10214/dinkel/dbms"
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
	})
	if err != nil {
		return err
	}
	d.driver = driver

	logrus.Debug("Setting up connection to the memgraph database")
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
		return err
	}
	return nil
}

// RunQuery runs the query against the memgraph DB and returns its result.
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

	logrus.Debug("Query finished")
	return queryResult
}

// GetSchema returns the database's current schema
func (d Driver) GetSchema(dbms.DBOptions) (*schema.Schema, error) {
	s := &schema.Schema{}
	s.Reset()
	return s, nil
}

// GetQueryResultType evaluates the produced result and returns the type the result indicates.
func (d Driver) GetQueryResultType(res dbms.QueryResult, errorMessageRegex *dbms.ErrorMessageRegex) dbms.QueryResultType {
	err := res.ProducedError
	if err == nil {
		return dbms.Valid
	}

	// https://github.com/neo4j/neo4j/issues/13101
	// For some reason the error for this is handled weirdly by the neo4j driver library, have to check for it like this
	if err.Error() == "expected success or database error" {
		return dbms.ReportedBug
	}

	switch err := err.(type) {
	case *neo4j.ConnectivityError:
		return dbms.Timeout
	case *neo4j.Neo4jError:
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

// DiscardQuery returns true with probability 1/3 or if the query produced a non-nil error,
// else it returns false. Thereby causing queries to have an expected amount of 4 statements
// if they don't produce an error.
func (d Driver) DiscardQuery(res dbms.QueryResult, seed *seed.Seed) bool {
	if res.ProducedError != nil {
		return true
	}

	out := seed.GetByte()
	return out%3 == 0
}

// VerifyConnectivity checks whether the DB is still reachable and hasn't crashed.
func (d Driver) VerifyConnectivity(dbms.DBOptions) (bool, error) {
	err := d.driver.VerifyConnectivity(context.Background())
	return err == nil, err
}

// IsEqualResult panics for memgraph, as it hasn't been implemented yet
func (d Driver) IsEqualResult(dbms.QueryResult, dbms.QueryResult) bool {
	panic("IsEqualResult is unimplemented for memgraph")
}
