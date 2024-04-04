// Code generated by "middlewarer -type=Strategy"; DO NOT EDIT.
package strategy

import (
	"context"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// WrapStrategy returns the passed Strategy wrapped in the middleware defined in StrategyMiddleware
func WrapStrategy(toWrap Strategy, wrapper StrategyMiddleware) Strategy {
	wrapper.wrapped = toWrap
	return &wrapper
}

// StrategyMiddleware implements Strategy
type StrategyMiddleware struct {
	wrapped Strategy

	DiscardQueryMiddleware             func(DiscardQueryHandler) DiscardQueryHandler
	GetQueryResultTypeMiddleware       func(GetQueryResultTypeHandler) GetQueryResultTypeHandler
	GetRootClauseMiddleware            func(GetRootClauseHandler) GetRootClauseHandler
	PrepareQueryForBugreportMiddleware func(PrepareQueryForBugreportHandler) PrepareQueryForBugreportHandler
	ReduceStepMiddleware               func(ReduceStepHandler) ReduceStepHandler
	ResetMiddleware                    func(ResetHandler) ResetHandler
	ValidateReductionResultMiddleware  func(ValidateReductionResultHandler) ValidateReductionResultHandler
	ValidateRerunResultsMiddleware     func(ValidateRerunResultsHandler) ValidateRerunResultsHandler
}

type DiscardQueryHandler func(dbms.QueryResultType, dbms.DB, dbms.DBOptions, dbms.QueryResult, *seed.Seed) bool
type GetQueryResultTypeHandler func(dbms.DB, dbms.DBOptions, dbms.QueryResult, *dbms.ErrorMessageRegex) dbms.QueryResultType
type GetRootClauseHandler func(translator.Implementation, *schema.Schema, *seed.Seed) translator.Clause
type PrepareQueryForBugreportHandler func([]string) []string
type ReduceStepHandler func(context.Context, []*helperclauses.ClauseCapturer) (context.Context, []*helperclauses.ClauseCapturer, bool)
type ResetHandler func()
type ValidateReductionResultHandler func(dbms.DB, []dbms.QueryResult, []dbms.QueryResult) bool
type ValidateRerunResultsHandler func([]dbms.QueryResult, dbms.DB) dbms.QueryResultType

func (s *StrategyMiddleware) DiscardQuery(a0 dbms.QueryResultType, a1 dbms.DB, a2 dbms.DBOptions, a3 dbms.QueryResult, a4 *seed.Seed) bool {
	fun := s.wrapped.DiscardQuery
	if s.DiscardQueryMiddleware != nil {
		fun = s.DiscardQueryMiddleware(fun)
	}
	return fun(a0, a1, a2, a3, a4)
}

func (s *StrategyMiddleware) GetQueryResultType(a0 dbms.DB, a1 dbms.DBOptions, a2 dbms.QueryResult, a3 *dbms.ErrorMessageRegex) dbms.QueryResultType {
	fun := s.wrapped.GetQueryResultType
	if s.GetQueryResultTypeMiddleware != nil {
		fun = s.GetQueryResultTypeMiddleware(fun)
	}
	return fun(a0, a1, a2, a3)
}

func (s *StrategyMiddleware) GetRootClause(a0 translator.Implementation, a1 *schema.Schema, a2 *seed.Seed) translator.Clause {
	fun := s.wrapped.GetRootClause
	if s.GetRootClauseMiddleware != nil {
		fun = s.GetRootClauseMiddleware(fun)
	}
	return fun(a0, a1, a2)
}

func (s *StrategyMiddleware) PrepareQueryForBugreport(a0 []string) []string {
	fun := s.wrapped.PrepareQueryForBugreport
	if s.PrepareQueryForBugreportMiddleware != nil {
		fun = s.PrepareQueryForBugreportMiddleware(fun)
	}
	return fun(a0)
}

func (s *StrategyMiddleware) ReduceStep(a0 context.Context, a1 []*helperclauses.ClauseCapturer) (context.Context, []*helperclauses.ClauseCapturer, bool) {
	fun := s.wrapped.ReduceStep
	if s.ReduceStepMiddleware != nil {
		fun = s.ReduceStepMiddleware(fun)
	}
	return fun(a0, a1)
}

func (s *StrategyMiddleware) Reset() {
	fun := s.wrapped.Reset
	if s.ResetMiddleware != nil {
		fun = s.ResetMiddleware(fun)
	}
	fun()
}

func (s *StrategyMiddleware) ValidateReductionResult(a0 dbms.DB, a1 []dbms.QueryResult, a2 []dbms.QueryResult) bool {
	fun := s.wrapped.ValidateReductionResult
	if s.ValidateReductionResultMiddleware != nil {
		fun = s.ValidateReductionResultMiddleware(fun)
	}
	return fun(a0, a1, a2)
}

func (s *StrategyMiddleware) ValidateRerunResults(a0 []dbms.QueryResult, a1 dbms.DB) dbms.QueryResultType {
	fun := s.wrapped.ValidateRerunResults
	if s.ValidateRerunResultsMiddleware != nil {
		fun = s.ValidateRerunResultsMiddleware(fun)
	}
	return fun(a0, a1)
}
