/*
Package redisgraph provides the model for RedisGraph, a Redis module.
*/
package redisgraph

import (
	"fmt"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	rg "github.com/RedisGraph/redisgraph-go"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

// Driver for RedisGraph
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

// RunQuery runs the query against the RedisGraph DB and returns its result.
func (d *Driver) RunQuery(opts dbms.DBOptions, query string) dbms.QueryResult {
	// TODO: Don't ignore result
	_, err := d.graph.ParameterizedQuery(query, map[string]any{"TIMEOUT_DEFAULT": fmt.Sprint(opts.Timeout.Seconds())})
	if err != nil {
		logrus.Debugf("Query produced error - %v", err)
	}
	return dbms.QueryResult{
		ProducedError: err,
	}
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

// IsEqualResult panics for redisgraph, as it hasn't been implemented yet
func (d *Driver) IsEqualResult(dbms.QueryResult, dbms.QueryResult) bool {
	panic("IsEqualResult is unimplemented for redisgraph")
}
