package none

import (
	"context"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/opencypher"
	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
	"github.com/sirupsen/logrus"
)

type Strategy struct{}

func (s Strategy) Reset() {}

func (s *Strategy) GetRootClause(impl translator.Implementation, schema *schema.Schema, seed *seed.Seed) translator.Clause {
	return &opencypher.RootClause{}
}

func (s *Strategy) GetQueryResultType(db dbms.DB, dbOpts dbms.DBOptions, res dbms.QueryResult, errorMessageRegex *dbms.ErrorMessageRegex) dbms.QueryResultType {
	return db.GetQueryResultType(res, errorMessageRegex)
}

func (s *Strategy) DiscardQuery(resultType dbms.QueryResultType, db dbms.DB, dbOpts dbms.DBOptions, res dbms.QueryResult, seed *seed.Seed) bool {
	return db.DiscardQuery(res, seed)
}

type noStrategyReductionContext int

const (
	// The index of the last root clause removed
	//
	// Starts at rootClauses - 1 and goes down until 0
	lastRemovedRootClauseIndex noStrategyReductionContext = iota
	// The index of the statement last reduced
	reducedStatementIndexNoStrategy
)

// Simply removes root clauses one by one
func (s *Strategy) ReduceStep(ctx context.Context, rootClauses []*helperclauses.ClauseCapturer) (context.Context, []*helperclauses.ClauseCapturer, bool) {
	if ctx.Value(lastRemovedRootClauseIndex) == nil {
		ctx = context.WithValue(ctx, lastRemovedRootClauseIndex, len(rootClauses)-2)
	}
	indexToRemove := ctx.Value(lastRemovedRootClauseIndex).(int)
	if indexToRemove >= 0 {
		return context.WithValue(ctx, lastRemovedRootClauseIndex, indexToRemove-1), append(rootClauses[:indexToRemove], rootClauses[indexToRemove+1:]...), false
	}

	if ctx.Value(reducedStatementIndexNoStrategy) == nil {
		ctx = context.WithValue(ctx, reducedStatementIndexNoStrategy, 0)
	}
	if index := ctx.Value(reducedStatementIndexNoStrategy).(int); index != len(rootClauses) {
		newCtx, isDone := s.reduceStatement(ctx, rootClauses[index])
		if isDone {
			// Reset context values not added in ReduceStep
			newCtx = context.WithValue(context.Background(), lastRemovedRootClauseIndex, -1)
			newCtx = context.WithValue(newCtx, reducedStatementIndexNoStrategy, index+1)
			logrus.Infof("Finished reducing statement #%d/%d", index+1, len(rootClauses))
		}
		return newCtx, rootClauses, false
	}

	return ctx, rootClauses, true
}

type noStrategyStatementReductionContext int

const (
	// The index of the last expression which was transformed to a property literal
	//
	// Starts at 0 and grows until all expressions got touched
	lastTransformedExpressionIndex noStrategyStatementReductionContext = iota
	// Set once all expressions got transformed once
	doneReducingExpressions
	// The index of the last clause transformed to an empty clause
	//
	// Starts at #nodes and shrinks until 0
	lastTransformedClauseIndex
)

// TODO: There has to be a better place to put this...
// NoStrategyReducible contains the NoStrategyReduce function.
//
// Clauses may implement this interface to make reduction more precise.
// Clauses not implementing this interface will get replaced by empty clauses for reduction.
type NoStrategyReducible interface {
	translator.Clause
	// This function gets called on clauses when reducing a bug report generated on
	// a bug which was found using no strategy.
	//
	// Gets passed the clause capturer that captured the clause.
	NoStrategyReduce(*helperclauses.ClauseCapturer) translator.Clause
}

// In a first phase, reduces by replacing every expression by a propertyLiteral of its
// underlying property type.
// Reduces further in a second phase by replacing every element by an EmptyClause.
func (s *Strategy) reduceStatement(ctx context.Context, rootClause *helperclauses.ClauseCapturer) (context.Context, bool) {
	if ctx.Value(doneReducingExpressions) == nil {
		if ctx.Value(lastTransformedExpressionIndex) == nil {
			ctx = context.WithValue(ctx, lastTransformedExpressionIndex, 0)
		}

		expressionIndex := ctx.Value(lastTransformedExpressionIndex).(int)
		if _, successful := transformExpressionByIndex(rootClause, expressionIndex); !successful {
			logrus.Info("Done transforming expressions")
			ctx = context.WithValue(ctx, doneReducingExpressions, true)
		}

		return context.WithValue(ctx, lastTransformedExpressionIndex, expressionIndex+1), false
	}

	if ctx.Value(lastTransformedClauseIndex) == nil {
		ctx = context.WithValue(ctx, lastTransformedClauseIndex, getNodesCount(rootClause))
	}

	targetIndex := ctx.Value(lastTransformedClauseIndex).(int)
	transformClauseAtIndex(rootClause, targetIndex)

	return context.WithValue(ctx, lastTransformedClauseIndex, targetIndex-1), targetIndex == 0
}

// Returns the amount of expressions contained in the subtree with the clause as its root.
//
// Returns true if a clause got transformed, else false.
//
// If the second return value is true, the first value will be arbitrary.
func transformExpressionByIndex(clause *helperclauses.ClauseCapturer, index int) (int, bool) {
	// Amount of expressions in the subtree
	var count int

	if asExp, ok := clause.GetCapturedClause().(*clauses.Expression); ok {
		// If this is the i-th expression
		if index == 0 {
			clause.UpdateClause(&clauses.PropertyLiteral{Conf: asExp.Conf})
			return 0, true
		}
		count++
	}
	for _, subclause := range clause.GetSubclauseClauseCapturers() {
		subCount, ok := transformExpressionByIndex(subclause, index-count)
		if ok {
			return 0, true
		}
		count += subCount
	}
	return count, false
}

// Returns the amount of nodes in the tree with the passed clause as the root
func getNodesCount(clause *helperclauses.ClauseCapturer) int {
	var count int
	for _, subclause := range clause.GetSubclauseClauseCapturers() {
		count += getNodesCount(subclause)
	}
	return count + 1
}

// Transforms the i-th clause to an EmptyClause
func transformClauseAtIndex(clause *helperclauses.ClauseCapturer, index int) {
	if index == 0 {
		if asReducable, ok := clause.GetCapturedClause().(NoStrategyReducible); ok {
			clause.UpdateClause(asReducable.NoStrategyReduce(clause))
		} else {
			clause.UpdateClause(&helperclauses.EmptyClause{})
		}
		return
	}
	index--
	for _, subclause := range clause.GetSubclauseClauseCapturers() {
		subclauseSize := getNodesCount(subclause)
		if subclauseSize > index {
			transformClauseAtIndex(subclause, index)
			return
		}
		index -= subclauseSize
	}
}

// Compares the errors produced by the last statements.
// Uses the driver's IsEqualResult function if implemented,
// otherwise it just compares the strings of the errors.
func (s *Strategy) ValidateReductionResult(driver dbms.DB, orig []dbms.QueryResult, new []dbms.QueryResult) bool {
	// Only care about last statement's error
	lastOrig := dbms.QueryResult{ProducedError: orig[len(orig)-1].ProducedError}
	lastNew := dbms.QueryResult{ProducedError: new[len(new)-1].ProducedError}

	if lastOrig.ProducedError == nil {
		return lastNew.ProducedError == nil
	} else if lastNew.ProducedError == nil {
		return false
	}
	return lastOrig.ProducedError.Error() == lastNew.ProducedError.Error()
}

func (s *Strategy) PrepareQueryForBugreport(query []string) []string {
	return query
}

func (s *Strategy) RerunQuery(statements []string, db dbms.DB, dbOpts dbms.DBOptions, runNext func() (dbms.QueryResult, error)) (dbms.QueryResultType, error) {
	for range statements {
		res, err := runNext()
		if err != nil {
			return dbms.Invalid, err
		}
		if res.Type != dbms.Valid {
			return res.Type, nil
		}
	}
	return dbms.Valid, nil
}
