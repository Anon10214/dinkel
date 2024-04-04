package prometheus

import (
	"context"
	"strings"
	"time"

	"github.com/Anon10214/dinkel/dbms"
	opencypherClauses "github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/scheduler/strategy"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
	"github.com/sirupsen/logrus"
)

func getStrategyHooks(exporter *dinkelExporter) strategy.StrategyMiddleware {
	return strategy.StrategyMiddleware{
		DiscardQueryMiddleware:  exporter.handleDiscardQuery,
		GetRootClauseMiddleware: exporter.handleGetRootClause,
	}
}

// handleDiscardQuery is the DiscardQuery handler for dinkelExporter.
// If the query is to be discarded, this handler increments the counter for the passed query result type.
func (e *dinkelExporter) handleDiscardQuery(next strategy.DiscardQueryHandler) strategy.DiscardQueryHandler {
	return func(resType dbms.QueryResultType, db dbms.DB, dbOpts dbms.DBOptions, res dbms.QueryResult, s *seed.Seed) bool {
		willDiscard := next(resType, db, dbOpts, res, s)

		e.statementCount.Inc()
		if willDiscard {
			e.queryCount.Inc()
			e.queryResultCounters[resType].Inc()
		}

		return willDiscard
	}
}

// handleGetRootClause is the GetRootClause handler for dinkelExporter.
// It sets the query generation start field of the exporter for calculating the generation latency in
// the RunQuery handler.
func (e *dinkelExporter) handleGetRootClause(next strategy.GetRootClauseHandler) strategy.GetRootClauseHandler {
	e.queryGenerationStart = time.Now()
	return next
}

func getFullStrategyHooks(exporter *fullDinkelExporter) strategy.StrategyMiddleware {
	return strategy.StrategyMiddleware{
		GetQueryResultTypeMiddleware: exporter.handleGetQueryResultType,
		GetRootClauseMiddleware:      exporter.handleGetRootClause,
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

		if err := e.analysisSemaphore.Acquire(context.Background(), 1); err != nil {
			logrus.Panicf("Failed to acquire analysis semaphore")
		}
		e.updateDataDependenciesCount(e.lastQueryClause)
		e.analysisSemaphore.Release(1)

		return res
	}
}

// updateDataDependenciesCount recursively goes through the AST whose root is the passed ClauseCapturer and
// updates the QC and AGS data dependencies count based on the underlying clause
func (e *fullDinkelExporter) updateDataDependenciesCount(clauseCapturer *helperclauses.ClauseCapturer) {

	switch clauseCapturer.GetCapturedClause().(type) {
	case *opencypherClauses.ExistingLabel: // Label taken from AGS
		if clauseCapturer.GetSubclauseClauseCapturers() == nil {
			// If ExistingLabel is a leaf of the AST,
			// it cannot have not defected to a new label
			// and thus induced a data dependency
			e.agsDataDependencies.Add(1)
		}

	case *opencypherClauses.ExistingProperty: // Property taken from AGS
		if clauseCapturer.TemplateString() != "%s" {
			// If the template string isn't %s, then ExistingProperty has used a property from the AGS
			e.agsDataDependencies.Add(1)
		}

	case *opencypherClauses.VariableExpression:
		// Check if it generated a variable from QC
		name := clauseCapturer.TemplateString()
		_, isPropVar := clauseCapturer.GetCapturedSchema().PropertyVariablesByName[name]
		_, isStructuralVar := clauseCapturer.GetCapturedSchema().StructuralVariablesByName[name]
		if isPropVar || isStructuralVar {
			e.qcDataDependencies.Add(1)
		}

		// Check if it generated a structural property access via AGS
		name = strings.TrimPrefix(clauseCapturer.TemplateString(), "%s.")
		if _, ok := clauseCapturer.GetCapturedSchema().PropertyTypeByName[name]; ok {
			e.agsDataDependencies.Add(1)
		}

	case *opencypherClauses.WriteTarget:
		if _, ok := clauseCapturer.GetSubclauseClauseCapturers()[0].GetCapturedClause().(*helperclauses.Stringer); ok {
			// If the subclause is a helperclauses.Stringer, then the write target used a variable from the QC
			e.qcDataDependencies.Add(1)
		}

	case *opencypherClauses.RemoveLabelExpression:
		// If the template string isn't just %s, then the clause has used a variable from the QC as a target
		if clauseCapturer.TemplateString() != "%s" {
			e.qcDataDependencies.Add(1)
		}

	case *opencypherClauses.SetLabelExpression:
		// If the template string isn't %s, it starts with a name from the QC
		if clauseCapturer.TemplateString() != "%s" {
			e.qcDataDependencies.Add(1)
		}

	case *opencypherClauses.SetPropertyExpression:
		// If the template string doesn't start with %s, it starts with a name from the QC
		if !strings.HasPrefix(clauseCapturer.TemplateString(), "%s") {
			e.qcDataDependencies.Add(1)
		}
	}

	for _, subclause := range clauseCapturer.GetSubclauseClauseCapturers() {
		e.updateDataDependenciesCount(subclause)
	}
}

func (e *fullDinkelExporter) handleGetRootClause(next strategy.GetRootClauseHandler) strategy.GetRootClauseHandler {
	return func(impl translator.Implementation, schema *schema.Schema, seed *seed.Seed) translator.Clause {
		rootClause := next(impl, schema, seed)
		e.lastQueryClause = helperclauses.GetClauseCapturerForClause(rootClause, impl)

		return e.lastQueryClause
	}
}
