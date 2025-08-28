package equivalencetransformation

import (
	"context"

	"github.com/Anon10214/dinkel/scheduler/strategy/none"
	"github.com/Anon10214/dinkel/translator/helperclauses"
	"github.com/sirupsen/logrus"
)

type reductionStep int

const (
	// Removed the statements that haven't been transformed
	reducedUntransformed reductionStep = iota
	// Removed the starting statements which might not have had an influence on the result
	reducedStarting
	// Reduced transformations by flipping them back to their original
	reducedTransformations
	// Reduced the clauses in the original and transformed statements simultaneously
	reducedEquivalenceClauses
)

func (s *Strategy) ReduceStep(ctx context.Context, rootClauses []*helperclauses.ClauseCapturer) (context.Context, []*helperclauses.ClauseCapturer, bool) {
	if ctx.Value(reducedUntransformed) == nil {
		// Remove untransformed statements, always succeeds
		clauses := len(rootClauses)
		transformedStatements := s.statementIndex + 1
		logrus.Info("Finished removing untransformed statements")
		return context.WithValue(ctx, reducedUntransformed, true), append(rootClauses[:transformedStatements], rootClauses[clauses-transformedStatements:]...), false
	}

	if ctx.Value(reducedStarting) == nil {
		ctx, rootClauses, done := s.reduceStartingStatements(ctx, rootClauses)
		if done {
			logrus.Info("Finished reducing starting statements")
			ctx = context.WithValue(ctx, reducedStarting, true)
		}
		return ctx, rootClauses, false
	}

	if ctx.Value(reducedTransformations) == nil {
		ctx, rootClauses, done := s.reduceTransformations(ctx, rootClauses)
		if done {
			logrus.Info("Finished reducing equivalence transformations")
			ctx = context.WithValue(ctx, reducedTransformations, true)
		}
		return ctx, rootClauses, false
	}

	if ctx.Value(reducedEquivalenceClauses) == nil {
		ctx, rootClauses, done := s.reduceEquivalenceClauses(ctx, rootClauses)
		if done {
			logrus.Info("Finished reducing equivalence clauses")
			ctx = context.WithValue(ctx, reducedEquivalenceClauses, true)
		}
		return ctx, rootClauses, false
	}

	// Reset context if reduction gets repeated
	ctx = context.WithValue(context.Background(), reducedUntransformed, true)

	return ctx, rootClauses, true
}

// --------- Start of ReduceStartingStatements ---------

type reduceStartingStatementsStep int

const reduceStartingStatementsIndex reduceStartingStatementsStep = iota

// Remove starting statements to see if they had an effect on the result of the final statement
func (s *Strategy) reduceStartingStatements(ctx context.Context, rootClauses []*helperclauses.ClauseCapturer) (context.Context, []*helperclauses.ClauseCapturer, bool) {
	statements := len(rootClauses) / 2
	if statements == 1 {
		return ctx, rootClauses, true
	}
	if ctx.Value(reduceStartingStatementsIndex) == nil {
		ctx = context.WithValue(ctx, reduceStartingStatementsIndex, statements-2)
	}
	indexToRemove := ctx.Value(reduceStartingStatementsIndex).(int)
	firstHalf := append(rootClauses[:indexToRemove], rootClauses[indexToRemove+1:statements]...)
	secondHalf := append(rootClauses[statements:statements+indexToRemove], rootClauses[statements+indexToRemove+1:]...)
	return context.WithValue(ctx, reduceStartingStatementsIndex, indexToRemove-1), append(firstHalf, secondHalf...), indexToRemove == 0
}

// --------- Start of ReduceTransformations ---------

type reduceTransformationsStep int

const (
	// The index of the statement last reduced
	reduceTransformationsIndex reduceTransformationsStep = iota
	// The index of the next transformed clause to flip in the current root clause
	reduceTransformationsFlipIndex
)

// Revert equivalence transformations of clauses
func (s *Strategy) reduceTransformations(ctx context.Context, rootClauses []*helperclauses.ClauseCapturer) (context.Context, []*helperclauses.ClauseCapturer, bool) {
	// Init context values if needed
	if ctx.Value(reduceTransformationsIndex) == nil {
		ctx = context.WithValue(ctx, reduceTransformationsIndex, 0)
		ctx = context.WithValue(ctx, reduceTransformationsFlipIndex, 0)
	}

	index := ctx.Value(reduceTransformationsIndex).(int)
	rootClause := rootClauses[index]
	flipIndex := ctx.Value(reduceTransformationsFlipIndex).(int)

	nextIndex, done := flipTransformed(flipIndex, rootClause)
	if !done {
		index++
		ctx = context.WithValue(ctx, reduceTransformationsIndex, index)
		ctx = context.WithValue(ctx, reduceTransformationsFlipIndex, 0)
		return ctx, rootClauses, index == len(rootClauses)
	}
	// Set index of next flip
	return context.WithValue(ctx, reduceTransformationsFlipIndex, nextIndex), rootClauses, false
}

// Traverses the AST in a DFS strategy, looking for the index of the next transformed clause to flip.
// Takes in the index of the transformed clause to flip, relative to the passed clause.
// Returns the amount of transformed clauses encountered and returns false if the index was not found, indicating that reduction was fully performed.
func flipTransformed(flipIndex int, clause *helperclauses.ClauseCapturer) (int, bool) {
	// How many transformed clauses were encountered
	encountered := 0
	if transformed, ok := clause.GetCapturedClause().(*TransformedClause); ok {
		// Flip the transformed clause if it is being used
		// Flip index may be negative since we may have encountered multiple untransformed clauses
		if flipIndex <= 0 && transformed.UseTransformed {
			transformedCopy := *transformed
			transformedCopy.UseTransformed = false
			clause.UpdateClause(&transformedCopy)
			return 1, true
		}
		encountered++
	}

	// Traverse subclauses
	for _, subclause := range clause.GetSubclauseClauseCapturers() {
		// Call flipTransformed for all subclauses
		var success bool
		subclauseEncountered, success := flipTransformed(flipIndex-encountered, subclause)
		encountered += subclauseEncountered
		if success {
			return encountered, true
		}
	}
	return encountered, false
}

// --------- Start of ReduceEquivalentClauses ---------

type reduceEquivalenceClausesStep int

const (
	// The index of the equivalence statements being reduced
	reduceEquivalenceStatementIndex reduceEquivalenceClausesStep = iota
	// The index of the clause of the equivalence statements reduced
	reduceEquivalenceClausesClauseIndex
)

// Remove clauses from statement i and statement i+transformedStatements simultaneously
func (s *Strategy) reduceEquivalenceClauses(ctx context.Context, rootClauses []*helperclauses.ClauseCapturer) (context.Context, []*helperclauses.ClauseCapturer, bool) {
	// Init context values if needed
	if ctx.Value(reduceEquivalenceStatementIndex) == nil {
		ctx = context.WithValue(ctx, reduceEquivalenceStatementIndex, 0)
		ctx = context.WithValue(ctx, reduceEquivalenceClausesClauseIndex, 0)
	}

	index := ctx.Value(reduceEquivalenceStatementIndex).(int)
	clauseIndex := ctx.Value(reduceEquivalenceClausesClauseIndex).(int)

	if _, ok := reduceClauseAtIndex(rootClauses[index], rootClauses[index+len(rootClauses)/2], clauseIndex); ok {
		ctx = context.WithValue(ctx, reduceEquivalenceClausesClauseIndex, clauseIndex+1)
	} else {
		index++
		ctx = context.WithValue(ctx, reduceEquivalenceClausesClauseIndex, 0)
		ctx = context.WithValue(ctx, reduceEquivalenceStatementIndex, index)
	}

	return ctx, rootClauses, index == len(rootClauses)/2
}

// reduceClauseAtIndex takes in the root clause of the original statement and the transformed one, as well as the index of the clause that should be removed, relative to the passed nodes.
// It returns the amount of nodes traversed and true if a node was removed, else false.
func reduceClauseAtIndex(orig, new *helperclauses.ClauseCapturer, index int) (int, bool) {
	if transformer, ok := new.GetCapturedClause().(*TransformedClause); ok {
		if transformer.UseTransformed {
			return 0, false
		}
		new = new.GetSubclauseClauseCapturers()[0]
	}

	// Remove this node
	if index == 0 {
		if asReducable, ok := orig.GetCapturedClause().(none.NoStrategyReducible); ok {
			orig.UpdateClause(asReducable.NoStrategyReduce(orig))
			new.UpdateClause(new.GetCapturedClause().(none.NoStrategyReducible).NoStrategyReduce(new))
		} else {
			orig.UpdateClause(&helperclauses.EmptyClause{})
			new.UpdateClause(&helperclauses.EmptyClause{})
		}
		return 0, true
	}

	nodesTraversed := 1
	origSubclauses := orig.GetSubclauseClauseCapturers()
	newSubclauses := new.GetSubclauseClauseCapturers()
	for i := range origSubclauses {
		offset, ok := reduceClauseAtIndex(origSubclauses[i], newSubclauses[i], index-nodesTraversed)
		nodesTraversed += offset
		if ok {
			return nodesTraversed, true
		}
	}

	return nodesTraversed, false
}
