/*
TODO: Docs
*/
package strategy

import (
	"context"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/scheduler/strategy/equivalencetransformation"
	"github.com/Anon10214/dinkel/scheduler/strategy/none"
	"github.com/Anon10214/dinkel/scheduler/strategy/predicatepartitioning"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
	"github.com/sirupsen/logrus"
)

// A FuzzingStrategy dictates how queries are generated and how bugs are detected.
type FuzzingStrategy int

const (
	// None represents no strategy, it just generates clauses randomly and sees if they trigger exception or crash bugs.
	None FuzzingStrategy = iota
	// EquivalenceTransformation is a strategy which first generates clauses,
	// and then transforms them to an equivalent clause and checks if they cause a different result.
	//
	// For example:
	//	null = null + x or
	//	MATCH (x)-[*5]-(y) = (x)-[*3]-()-[*2]-(y)
	EquivalenceTransformation
	// PredicatePartitioning uses methods introduced by Manuel Rigger:
	//
	//	MATCH (..) RETURN ..
	//
	// must return the same rows as
	//
	//	MATCH (..) WHERE x RETURN ..
	//	 UNION ALL
	//	MATCH (..) WHERE NOT x RETURN ..
	//	 UNION ALL
	//	MATCH (..) WHERE x IS NULL RETURN ..
	PredicatePartitioning
)

// ToStrategy returns the concrete [Strategy] associated with a [FuzzingStrategy].
func (s FuzzingStrategy) ToStrategy() Strategy {
	switch s {
	case None:
		return &none.Strategy{}
	case EquivalenceTransformation:
		return &equivalencetransformation.Strategy{}
	case PredicatePartitioning:
		return &predicatepartitioning.Strategy{}
	}
	logrus.Panicf("Invalid Fuzzing strategy encountered: %d", s)
	return nil
}

// ToString converts a fuzzing strategy to its equivalent, human-readable string representation
func (s FuzzingStrategy) ToString() string {
	switch s {
	case None:
		return "NONE"
	case EquivalenceTransformation:
		return "EQUIVALENCE TRANSFORM"
	case PredicatePartitioning:
		return "PREDICATE PARTITIONING"
	}
	return "INVALID FUZZING STRATEGY"
}

// The Strategy interface represents a fuzzing strategy.
//
// A strategy dictates how queries are generated, which results indicate bugs and how queries are reduced.
// Available strategies can be viewed at [FuzzingStrategy].
//
//go:generate middlewarer -type=Strategy
type Strategy interface {
	// Reset the strategy to prepare for another fuzzing run
	Reset()
	GetRootClause(translator.Implementation, *schema.Schema, *seed.Seed) translator.Clause
	GetQueryResultType(dbms.DB, dbms.DBOptions, dbms.QueryResult, *dbms.ErrorMessageRegex) dbms.QueryResultType
	DiscardQuery(dbms.QueryResultType, dbms.DB, dbms.DBOptions, dbms.QueryResult, *seed.Seed) bool
	// Reduces the queries from a bug report.
	//
	// Gets passed a context which will be preserved between calls to the reduce function.
	// The second argument is the slice of the currently most reduced root clauses composing the queries.
	//
	// Returns the new context, the newly most reduced queries and true if the reduction is done or false if it can be further reduced.
	//
	// After every reduce call, the passed clause capturer will be regenerated and run against the target.
	// If the query's result now differs, the next ReduceStep call will have the previous root clauses as the second argument.
	// If the query's result still matches, the next ReduceStep call will have the new root clauses as the second argument.
	ReduceStep(context.Context, []*helperclauses.ClauseCapturer) (context.Context, []*helperclauses.ClauseCapturer, bool)
	// Gets passed the results of the original and reduced queries.
	// Returns true if the results still point to the same bug being present, else false.
	ValidateReductionResult(dbms.DB, []dbms.QueryResult, []dbms.QueryResult) bool
	// Gets called when a bug is found with the query generated.
	//
	// Useful for when some statements are not relevant for the bug report.
	//
	// For example during equivalence transformation, if the bug was found before
	// fully transforming all statements, the last few, untransformed, statements can just be omitted.
	PrepareQueryForBugreport([]string) []string
	// RerunQuery receives the statements to be rerun, variables needed by the strategy for running queries
	// in addition to a function which runs the next query and returns the query result received when running the next statement and an optional error.
	//
	// This function returns the result type that the full query indicates and an optional error.
	//
	// This allows strategies to perform things like resetting the database if needed during a rerun.
	RerunQuery(statements []string, db dbms.DB, dbOpts dbms.DBOptions, runNext func() (dbms.QueryResult, error)) (dbms.QueryResultType, error)
}
