// Package dbms provides types and constants for database interaction.
package dbms

import (
	"regexp"
	"time"

	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
)

// QueryResult embodies the result returned from a DBMS after running a query
type QueryResult struct {
	// The type this query result indicates
	Type QueryResultType
	// Returned rows
	Rows []any
	// The error as returned by the driver
	ProducedError error
	// The fingerprint of the graph, used for comparing if results changed
	Fingerprint string
	// The DB schema
	Schema any
}

// A QueryResultType specifies what a query's result indicates to dictate how to classify the query
type QueryResultType int

const (
	// None indicates the result type was not set yet
	None QueryResultType = iota
	// Valid indicates query of no importance was run
	Valid
	// Invalid indicates a semantically or syntactically invalid query was run
	Invalid
	// Bug indicates that the query triggered a bug in the DBMS
	Bug
	// Crash indicates that the query caused the DBMS to crash
	Crash
	// ReportedBug indicates that the query triggered a bug in the DBMS which has already been reported
	ReportedBug
	// Timeout indicates that the query timed out
	Timeout
)

// ToString converts a query result type to its equivalent, human-readable string representation
func (q QueryResultType) ToString() string {
	switch q {
	case Valid:
		return "VALID"
	case Invalid:
		return "INVALID"
	case Bug:
		return "BUG"
	case Crash:
		return "CRASH"
	case ReportedBug:
		return "REPORTED_BUG"
	case Timeout:
		return "TIMEOUT"
	}
	return "UNDEFINED QUERY RESULT TYPE"
}

// DBOptions specify driver behavior and connection information.
type DBOptions struct {
	// The host where the DB is accessible at.
	Host string
	// The port where the DB is accessible at.
	//
	// If the port is nil, the driver should use the default port for the DBMS.
	Port *int
	// The timeout for database requests.
	// This timeout should be used for every request that is sent to the DB.
	Timeout time.Duration
	// Whether the driver should run in backwards compatible mode.
	// This is used during bisection, where older versions may be tested and some features either disabled or adjusted
	BackwardsCompatibleMode bool
}

// ErrorMessageRegex holds regular expressions, matching different types of error messages
type ErrorMessageRegex struct {
	// Ignored matches error messages that should be ignored
	Ignored *regexp.Regexp
	// Reported matches error messages that indicate an already reported bug
	Reported *regexp.Regexp
}

// A DB is a database driver implementation.
//
// Any struct implementing this interface can be used as a fuzzing target.
//
//go:generate middlewarer -type=DB
type DB interface {
	// Initialises the DB connection
	Init(DBOptions) error
	// Reset the DB to its original state
	Reset(DBOptions) error
	// Fetch the DB's schema
	GetSchema(DBOptions) (*schema.Schema, error)
	// Runs a given query against the database
	RunQuery(DBOptions, string) QueryResult
	// Verifies that the database can still be reached.
	//
	// Returns true if connection is successful and an optional error describing the connection error.
	//
	// If, after a query was run, the database is no longer reachable,
	// a bug report is created with the assumption that the database crashed
	VerifyConnectivity(DBOptions) (bool, error)

	// Returns the type of the query's result.
	GetQueryResultType(QueryResult, *ErrorMessageRegex) QueryResultType
	// Returns true if the query should not be generated further.
	// Gets passed the result of executing the previous statement and the seed being used for generation
	DiscardQuery(QueryResult, *seed.Seed) bool
	// Returns true if the two passed query results are equal, else false.
	// Used by strategies fuzzing for logic bugs.
	IsEqualResult(QueryResult, QueryResult) bool
}
