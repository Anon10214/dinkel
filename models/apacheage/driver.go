/*
Package apacheage provides the model for Apache AGE, a postgres extension.
*/
package apacheage

import (
	"database/sql"
	"fmt"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/apache/age/drivers/golang/age"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Driver for apache age
type Driver struct {
	driver *sql.DB
}

// Init the DB driver
func (d *Driver) Init(opts dbms.DBOptions) error {
	connPort := 5432
	if opts.Port != nil {
		connPort = *opts.Port
	}

	var err error
	if d.driver, err = sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=postgres sslmode=disable", opts.Host, connPort)); err != nil {
		return err
	}

	logrus.Debug("Setting up connection to the apache age database")
	return nil
}

// initAgeTransaction runs the boilerplate statements for initializing an apache age transaction.
func (d *Driver) initAgeTransaction() (*sql.Tx, error) {
	tx, err := d.driver.Begin()
	if err != nil {
		return nil, err
	}

	// Apache age boilerplate
	for _, query := range []string{
		`LOAD 'age';`,
		`SET search_path = ag_catalog, "$user", public;`,
	} {
		if _, err := tx.Exec(query); err != nil {
			return nil, err
		}
	}
	return tx, nil
}

// Reset the database
func (d *Driver) Reset(opts dbms.DBOptions) error {
	logrus.Debug("Resetting Database")

	// Drop the graph in a separate transaction, as when this functions is called for
	// the first time, it will error as no graph exists yet.
	tx, err := d.initAgeTransaction()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`SELECT drop_graph('graph', true);`); err != nil {
		if err.Error() != `pq: graph "graph" does not exist` {
			tx.Rollback()
			return err
		}
	} else if err := tx.Commit(); err != nil {
		return err
	}

	// Create a new graph
	tx, err = d.initAgeTransaction()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`SELECT create_graph('graph');`); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// RunQuery runs the query against the apache age DB and returns its result.
func (d Driver) RunQuery(opts dbms.DBOptions, query string) dbms.QueryResult {
	res := dbms.QueryResult{}

	tx, err := d.initAgeTransaction()
	if err != nil {
		res.ProducedError = err
		tx.Rollback()
		return res
	}

	if _, err := tx.Exec(fmt.Sprintf(`SELECT * FROM cypher('graph',$$
	%s
$$) as (v agtype);`, query)); err != nil {
		res.ProducedError = err
		tx.Rollback()
		return res
	}

	res.ProducedError = tx.Commit()

	// Get schema
	cursor, err := age.ExecCypher(tx, "graph", 1, "MATCH (n) RETURN n AS x UNION MATCH ()-[m]-() RETURN m AS x")
	if err != nil {
		res.ProducedError = err
		logrus.Debugf("Error %v produced when trying to get schema", err)
		return res
	}

	for cursor.Next() {
		fmt.Println(cursor.GetRow())
	}

	return res
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

	switch err := err.(type) {
	case *pq.Error:
		if errorMessageRegex.Ignored.MatchString(err.Message) {
			return dbms.Invalid
		}

		if errorMessageRegex.Reported.MatchString(err.Message) {
			return dbms.ReportedBug
		}

		logrus.Warnf("Encountered pqError with error code %s and msg %s", err.Code, err.Message)
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
	err := d.driver.Ping()
	return err == nil, err
}

// IsEqualResult panics for apache age, as it hasn't been implemented yet
func (d Driver) IsEqualResult(dbms.QueryResult, dbms.QueryResult) bool {
	panic("IsEqualResult is unimplemented for apache AGE")
}
