/*
Package mock provides a mock implementation and driver for testing purposes.
*/
package mock

import (
	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
)

// Driver for the mock model
type Driver struct{}

// Init does nothing and returns nil
func (d *Driver) Init(opts dbms.DBOptions) error {
	return nil
}

// Reset does nothing and returns nil
func (d *Driver) Reset(opts dbms.DBOptions) error {
	return nil
}

// RunQuery does nothing and returns an empty query result
func (d Driver) RunQuery(opts dbms.DBOptions, query string) dbms.QueryResult {
	return dbms.QueryResult{}
}

// GetSchema does nothing and returns a default, initialized schema and nil
func (d Driver) GetSchema(opts dbms.DBOptions) (*schema.Schema, error) {
	s := &schema.Schema{}
	s.Reset()
	return s, nil
}

// GetQueryResultType always returns [dbms.Valid]
func (d Driver) GetQueryResultType(res dbms.QueryResult) dbms.QueryResultType {
	return dbms.Valid
}

// DiscardQuery always returns true
func (d Driver) DiscardQuery(res dbms.QueryResult, seed *seed.Seed) bool {
	return true
}

// VerifyConnectivity always returns true and nil
func (d Driver) VerifyConnectivity(dbms.DBOptions) (bool, error) {
	return true, nil
}

// IsEqualResult always returns true
func (d Driver) IsEqualResult(a, b dbms.QueryResult) bool {
	return true
}
