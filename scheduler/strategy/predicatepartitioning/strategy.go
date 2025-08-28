package predicatepartitioning

import (
	"context"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type Strategy struct {
	generatedSchema              bool // If done generating the schema to be queried and tested via query partitioning
	generatedOriginalQuery       bool // If done generating the original query (MATCH (...) RETURN ..)
	generatedPartitioningQueries bool // If done generating the partitioning queries. Query can be discarded once this is true

	originalQueryResult dbms.QueryResult

	// Capturers for the partitioning part
	matchCapturer           *helperclauses.ClauseCapturer
	whereExpressionCapturer *helperclauses.ClauseCapturer
	returnCapturer          *helperclauses.ClauseCapturer
}

func (s *Strategy) Reset() {
	*s = Strategy{}
}

func (s *Strategy) GetRootClause(impl translator.Implementation, sc *schema.Schema, seed *seed.Seed) translator.Clause {
	helperclauses.SetImplementation(impl)
	if !s.generatedSchema {
		if out := seed.GetByte(); out%5 == 0 {
			s.generatedSchema = true
		}
		if out := seed.GetByte(); out%5 == 0 {
			return &clauses.Index{}
		}
		return &clauses.WriteClause{}
	} else if !s.generatedOriginalQuery {
		s.generatedOriginalQuery = true

		s.matchCapturer = helperclauses.GetClauseCapturerForClause(&clauses.PathPatternExpression{})
		s.whereExpressionCapturer = helperclauses.GetClauseCapturerForClause(&clauses.WhereExpression{})
		s.returnCapturer = helperclauses.GetClauseCapturerForClause(&clauses.Return{})

		sc.DisallowAggregateFunctions = true

		// Where expression has to be evaluated already too but ignore it in the template string
		return helperclauses.CreateAssembler(
			"MATCH %s %[3]s",
			s.matchCapturer, s.whereExpressionCapturer, s.returnCapturer,
		)
	}
	s.generatedPartitioningQueries = true
	// Return the query part thrice
	queryPart := []translator.Clause{s.matchCapturer, s.whereExpressionCapturer, s.returnCapturer}
	return helperclauses.CreateAssembler(
		`MATCH %s
WHERE %s
%s
	UNION ALL
MATCH %[1]s
WHERE NOT (%s)
%s
	UNION ALL
MATCH %[1]s
WHERE (%s) IS NULL
%s`,
		append(append(queryPart, queryPart...), queryPart...)...,
	)
}

func (s *Strategy) GetQueryResultType(db dbms.DB, dbOpts dbms.DBOptions, res dbms.QueryResult, errorMessageRegex *dbms.ErrorMessageRegex) dbms.QueryResultType {
	if resType := db.GetQueryResultType(res, errorMessageRegex); resType != dbms.Valid {
		return resType
	}

	if s.generatedOriginalQuery && !s.generatedPartitioningQueries {
		s.originalQueryResult = res
	}

	if !s.generatedPartitioningQueries {
		return dbms.Valid
	}

	if !db.IsEqualResult(s.originalQueryResult, res) {
		return dbms.Bug
	}

	return dbms.Valid
}

func (s *Strategy) DiscardQuery(resultType dbms.QueryResultType, db dbms.DB, dbOpts dbms.DBOptions, res dbms.QueryResult, seed *seed.Seed) bool {
	return db.DiscardQuery(res, seed) || s.generatedPartitioningQueries
}

// TODO
func (s *Strategy) ReduceStep(ctx context.Context, rootClauses []*helperclauses.ClauseCapturer) (context.Context, []*helperclauses.ClauseCapturer, bool) {
	return ctx, rootClauses, true
}

// TODO
func (s *Strategy) ValidateReductionResult(dbms.DB, []dbms.QueryResult, []dbms.QueryResult) bool {
	return true
}

func (s *Strategy) PrepareQueryForBugreport(query []string) []string {
	return query
}

func (s *Strategy) RerunQuery(statements []string, db dbms.DB, dbOpts dbms.DBOptions, runNext func() (dbms.QueryResult, error)) (dbms.QueryResultType, error) {
	panic("Rerun Query is not yet implemented for predicate partitioning strategy")
}
