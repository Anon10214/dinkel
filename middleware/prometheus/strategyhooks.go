package prometheus

import (
	"context"
	"strings"
	"time"

	"github.com/Anon10214/dinkel/dbms"
	opencypherClauses "github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/scheduler/strategy"
	"github.com/Anon10214/dinkel/scheduler/strategy/equivalencetransformation"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
	"github.com/sirupsen/logrus"
)

// getUnderlyingDataDependencies recursively goes through the AST whose root is the passed ClauseCapturer and
// returns the QC and AGS data dependencies count based on the underlying clause
func getUnderlyingDataDependencies(clauseCapturer *helperclauses.ClauseCapturer) (int, int) {
	qcDeps, agsDeps := 0, 0

	switch clauseCapturer.GetCapturedClause().(type) {
	case *opencypherClauses.ExistingLabel: // Label taken from AGS
		if clauseCapturer.GetSubclauseClauseCapturers() == nil {
			// If ExistingLabel is a leaf of the AST,
			// it cannot have not defected to a new label
			// and thus induced a data dependency
			agsDeps += 1
		}

	case *opencypherClauses.ExistingProperty: // Property taken from AGS
		if clauseCapturer.TemplateString() != "%s" {
			// If the template string isn't %s, then ExistingProperty has used a property from the AGS
			agsDeps += 1
		}

	case *opencypherClauses.VariableExpression:
		// Check if it generated a variable from QC
		name := clauseCapturer.TemplateString()
		_, isPropVar := clauseCapturer.GetCapturedSchema().PropertyVariablesByName[name]
		_, isStructuralVar := clauseCapturer.GetCapturedSchema().StructuralVariablesByName[name]
		if isPropVar || isStructuralVar {
			qcDeps += 1
		}

		// Check if it generated a structural property access via AGS
		name = strings.TrimPrefix(clauseCapturer.TemplateString(), "%s.")
		if _, ok := clauseCapturer.GetCapturedSchema().PropertyTypeByName[name]; ok {
			agsDeps += 1
		}

	case *opencypherClauses.WriteTarget:
		if _, ok := clauseCapturer.GetSubclauseClauseCapturers()[0].GetCapturedClause().(*helperclauses.Stringer); ok {
			// If the subclause is a helperclauses.Stringer, then the write target used a variable from the QC
			qcDeps += 1
		}

	case *opencypherClauses.RemoveLabelExpression:
		// If the template string isn't just %s, then the clause has used a variable from the QC as a target
		if clauseCapturer.TemplateString() != "%s" {
			qcDeps += 1
		}

	case *opencypherClauses.SetLabelExpression:
		// If the template string isn't %s, it starts with a name from the QC
		if clauseCapturer.TemplateString() != "%s" {
			qcDeps += 1
		}

	case *opencypherClauses.SetPropertyExpression:
		// If the template string doesn't start with %s, it starts with a name from the QC
		if !strings.HasPrefix(clauseCapturer.TemplateString(), "%s") {
			qcDeps += 1
		}

	case *opencypherClauses.PropertyName:
		if clauseCapturer.GetCapturedClause().(*opencypherClauses.PropertyName).UseExstingName {
			agsDeps += 1
		}
	}

	for _, subclause := range clauseCapturer.GetSubclauseClauseCapturers() {
		subQcDeps, subAgsDeps := getUnderlyingDataDependencies(subclause)

		qcDeps += subQcDeps
		agsDeps += subAgsDeps
	}

	return qcDeps, agsDeps
}

func getStrategyHooks(exporter *dinkelExporter) strategy.StrategyMiddleware {
	return strategy.StrategyMiddleware{
		GetQueryResultTypeMiddleware: exporter.handleGetQueryResultType,
		GetRootClauseMiddleware:      exporter.handleGetRootClause,
	}
}

// handleGetQueryResultType is the DiscardQuery handler for dinkelExporter.
// If the query is to be discarded, this handler increments the counter for the passed query result type.
func (e *dinkelExporter) handleGetQueryResultType(next strategy.GetQueryResultTypeHandler) strategy.GetQueryResultTypeHandler {
	return func(db dbms.DB, dbOpts dbms.DBOptions, qr dbms.QueryResult, emr *dbms.ErrorMessageRegex) dbms.QueryResultType {
		res := next(db, dbOpts, qr, emr)

		e.queryCount.Inc()
		e.queryResultCounters[res].Inc()

		return res
	}
}

// handleGetRootClause is the GetRootClause handler for dinkelExporter.
// It sets the query generation start field of the exporter for calculating the generation latency in
// the RunQuery handler.
func (e *dinkelExporter) handleGetRootClause(next strategy.GetRootClauseHandler) strategy.GetRootClauseHandler {
	return func(i translator.Implementation, s1 *schema.Schema, s2 *seed.Seed) translator.Clause {
		clause := next(i, s1, s2)
		e.queryGenerationStart = time.Now()
		return clause
	}
}

func getFullStrategyHooks(exporter *fullDinkelExporter) strategy.StrategyMiddleware {
	return strategy.StrategyMiddleware{
		GetQueryResultTypeMiddleware: exporter.handleGetQueryResultType,
		GetRootClauseMiddleware:      exporter.handleGetRootClause,
	}
}

func (e *fullDinkelExporter) handleGetRootClause(next strategy.GetRootClauseHandler) strategy.GetRootClauseHandler {
	return func(impl translator.Implementation, schema *schema.Schema, seed *seed.Seed) translator.Clause {
		rootClause := next(impl, schema, seed)
		e.lastQueryClause = helperclauses.GetClauseCapturerForClause(rootClause)

		return e.lastQueryClause
	}
}

func (e *fullDinkelExporter) handleGetQueryResultType(next strategy.GetQueryResultTypeHandler) strategy.GetQueryResultTypeHandler {
	return func(db dbms.DB, dbOpts dbms.DBOptions, qr dbms.QueryResult, emr *dbms.ErrorMessageRegex) dbms.QueryResultType {
		res := next(db, dbOpts, qr, emr)

		e.querySize.Add(float64(len(e.lastQueryString)))

		for _, keyword := range e.keywords {
			count := float64(strings.Count(e.lastQueryString, keyword))
			e.totalKeywordCount.Add(count)
			e.keywordCount[keyword].Add(count)
		}

		go func(lastQueryClause *helperclauses.ClauseCapturer) {
			if err := e.analysisSemaphore.Acquire(context.Background(), 1); err != nil {
				logrus.Panicf("Failed to acquire analysis semaphore")
			}
			qcDeps, agsDeps := getUnderlyingDataDependencies(lastQueryClause)
			e.qcDataDependencies.Add(float64(qcDeps))
			e.agsDataDependencies.Add(float64(agsDeps))
			e.analysisSemaphore.Release(1)
		}(e.lastQueryClause)

		return res
	}
}

func getEquivalenceTransformationStrategyHooks(exporter *equivalenceTransformationDinkelExporter) strategy.StrategyMiddleware {
	return strategy.StrategyMiddleware{
		GetRootClauseMiddleware:      exporter.handleGetRootClause,
		GetQueryResultTypeMiddleware: exporter.handleGetQueryResultType,
	}
}

func (e *equivalenceTransformationDinkelExporter) handleGetRootClause(next strategy.GetRootClauseHandler) strategy.GetRootClauseHandler {
	return func(impl translator.Implementation, schema *schema.Schema, seed *seed.Seed) translator.Clause {
		rootClause := next(impl, schema, seed)
		e.lastQueryClause = helperclauses.GetClauseCapturerForClause(rootClause)

		return e.lastQueryClause
	}
}

// getTransformationStats traverses the AST whose root is the passed clause and returns (amount of clauses transformed, added QC dependencies, added AGS dependencies)
func getTransformationStats(clause *helperclauses.ClauseCapturer) (int, int, int) {
	transformedClauses, addedQCDependencies, addedAGSDependencies := 0, 0, 0

	var helper func(curClause *helperclauses.ClauseCapturer, isInTransformedAST bool)
	helper = func(curClause *helperclauses.ClauseCapturer, isInTransformedAST bool) {
		if transformationClause, ok := curClause.GetCapturedClause().(*equivalencetransformation.TransformedClause); ok {
			if transformationClause.UseTransformed {
				transformedClauses++
				// Update QC and AGS if this is the root of a transformed AST
				if !isInTransformedAST {
					subQCDependencies, subAGSDependencies := getUnderlyingDataDependencies(curClause)
					addedQCDependencies += subQCDependencies
					addedAGSDependencies += subAGSDependencies
				}
				isInTransformedAST = true
			}
		}

		for _, subclause := range curClause.GetSubclauseClauseCapturers() {
			helper(subclause, isInTransformedAST)
		}
	}

	helper(clause, false)

	return transformedClauses, addedQCDependencies, addedAGSDependencies
}

func (e *equivalenceTransformationDinkelExporter) handleGetQueryResultType(next strategy.GetQueryResultTypeHandler) strategy.GetQueryResultTypeHandler {
	return func(db dbms.DB, dbOpts dbms.DBOptions, qr dbms.QueryResult, emr *dbms.ErrorMessageRegex) dbms.QueryResultType {
		res := next(db, dbOpts, qr, emr)

		go func(lastQueryClause *helperclauses.ClauseCapturer) {
			if err := e.analysisSemaphore.Acquire(context.Background(), 1); err != nil {
				logrus.Panicf("Failed to acquire analysis semaphore")
			}
			transformedClauses, addedQcDeps, addedAgsDeps := getTransformationStats(lastQueryClause)
			if transformedClauses > 0 {
				e.transformedQueries.Inc()
				e.transformedQueriesResultCounters[res].Inc()
			}
			e.transformedClauses.Add(float64(transformedClauses))
			e.addedQcDataDependencies.Add(float64(addedQcDeps))
			e.addedAgsDataDependencies.Add(float64(addedAgsDeps))

			e.analysisSemaphore.Release(1)
		}(e.lastQueryClause)

		return res
	}
}
