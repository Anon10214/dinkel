package equivalencetransformation

import (
	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/opencypher"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
	"github.com/sirupsen/logrus"
)

type Strategy struct {
	statementsToGenerate int
	generatedStatements  []*helperclauses.ClauseCapturer
	previousResults      []dbms.QueryResult

	// If this strategy is currently in the process of transforming the statements
	isTransforming bool

	// The index of the statement we're at.
	//  0 <= statementIndex <= 2 * len(generatedClauses)
	// Gets incremented in the DiscardQuery function call
	statementIndex int
}

func (s *Strategy) Reset() {
	*s = Strategy{}
}

func (s *Strategy) GetRootClause(impl translator.Implementation, schema *schema.Schema, seed *seed.Seed) translator.Clause {
	helperclauses.SetImplementation(impl)
	schema.DisallowReturnAll = true
	if !s.isTransforming {
		// Generate a new statement
		s.statementsToGenerate++
		capturer := helperclauses.GetClauseCapturerForClause(&opencypher.RootClause{})
		s.generatedStatements = append(s.generatedStatements, capturer)
	} else if s.isTransforming {
		// Transform the current statement
		// Copy so it doesn't change the previous root clauses
		s.generatedStatements[s.statementIndex] = s.generatedStatements[s.statementIndex].Copy()
		equivalenceTransform(impl, s.generatedStatements[s.statementIndex], seed)
	}

	// Return the next clause in the generatedClauses list
	return s.generatedStatements[s.statementIndex]
}

func (s *Strategy) GetQueryResultType(db dbms.DB, dbOpts dbms.DBOptions, res dbms.QueryResult, errorMessageRegex *dbms.ErrorMessageRegex) dbms.QueryResultType {
	s.previousResults = append(s.previousResults, res)

	// If not transforming or result type is not VALID, return what driver returns
	if resType := db.GetQueryResultType(res, errorMessageRegex); !s.isTransforming || resType != dbms.Valid {
		return resType
	}

	if !db.IsEqualResult(s.previousResults[s.statementIndex], res) {
		return dbms.Bug
	}
	return dbms.Valid
}

func (s *Strategy) DiscardQuery(resultType dbms.QueryResultType, db dbms.DB, dbOpts dbms.DBOptions, res dbms.QueryResult, seed *seed.Seed) bool {
	// Transform valid queries if generated query is invalid and we haven't transformed yet
	if resultType == dbms.Invalid && !s.isTransforming && s.statementIndex != 0 {
		s.isTransforming = true
		s.statementsToGenerate = s.statementIndex
		s.generatedStatements = s.generatedStatements[:len(s.generatedStatements)-1]
		s.previousResults = s.previousResults[:len(s.previousResults)-1]
		s.statementIndex = 0
		db.Reset(dbOpts)
		return false
	} else if resultType != dbms.Valid {
		return true
	}

	s.statementIndex++

	// Reset the database if original statements have been generated
	if !s.isTransforming && db.DiscardQuery(res, seed) {
		// Generated clauses once, have to transform now, generate and compare results
		s.isTransforming = true
		s.statementIndex = 0
		// Reset DB
		db.Reset(dbOpts)
	}

	// Discard if the produced error is not nil or the clauses were transformed and generated
	return res.ProducedError != nil || (s.statementIndex == s.statementsToGenerate && s.isTransforming)
}

func (s *Strategy) ValidateReductionResult(driver dbms.DB, orig []dbms.QueryResult, new []dbms.QueryResult) bool {
	if len(orig) != len(new) {
		if (orig[len(orig)-1].ProducedError != nil) != (new[len(new)-1].ProducedError != nil) {
			return false
		}
	} else {
		for i, res := range orig {
			if (res.ProducedError != nil) != (new[i].ProducedError != nil) {
				return false
			}
		}
	}
	lastTransformedOld, lastTransformedNew := orig[len(orig)-1], new[len(new)-1]
	lastOriginalNew := new[len(new)/2-1]
	if lastTransformedOld.ProducedError != nil || lastTransformedNew.ProducedError != nil {
		if (lastTransformedOld.ProducedError == nil || lastTransformedNew.ProducedError == nil) || (lastTransformedOld.ProducedError.Error() != lastTransformedNew.ProducedError.Error()) {
			logrus.Info("Errors no longer match - reduction unsuccessful")
			return false
		}
		logrus.Info("Errors still matching and both not nil - reduction successful")
		return true
	}
	if driver.IsEqualResult(lastOriginalNew, lastTransformedNew) {
		logrus.Info("Last original result matches last transformed result - reduction unsuccessful")
		return false
	}
	logrus.Info("Last original result still doesn't match last transformed result - reduction successful")
	return true
}

func (s *Strategy) PrepareQueryForBugreport(query []string) []string {
	statements := len(query)
	// Discard original queries which weren't involved when triggering the bug
	return append(query[:s.statementIndex+1], query[statements-s.statementIndex-1:]...)
}

// RerunQuery reruns the query composed of the passed statements by repeatedly invoking the runNext function.
//
// If the amount of statements is odd, this function immediately returns a result indicating an invalid query.
// Otherwise, the query is rerun, the database is reset after processing half of all queries, and the gathered query results are compared.
// If the first half of results don't correspond to the second half, a bug is identified, otherwise, the query is valid.
func (s *Strategy) RerunQuery(statements []string, db dbms.DB, dbOpts dbms.DBOptions, runNext func() (dbms.QueryResult, error)) (dbms.QueryResultType, error) {
	if len(statements)%2 != 0 {
		logrus.Warnf("Amount of results received is odd (%d). Aborting comparison of query results.", len(statements))
		return dbms.Invalid, nil
	}

	results := []dbms.QueryResult{}

	// Run the statements
	for i := range statements {
		// Reset DB after running half of all statements
		if i == len(statements)/2 {
			if err := db.Reset(dbOpts); err != nil {
				return dbms.Invalid, err
			}
		}

		// Run next query
		res, err := runNext()
		if err != nil {
			return dbms.Invalid, err
		}
		if res.Type != dbms.Valid {
			return res.Type, err
		}

		results = append(results, res)
	}

	// Compare query results
	for i := 0; i < len(statements)/2; i++ {
		if !db.IsEqualResult(results[i], results[i+len(statements)/2]) {
			logrus.Warnf("Query #%d produced non-matching query results after equivalence tramsforming query #%d", i+len(statements)/2+1, i+1)
			return dbms.Bug, nil
		}
	}
	return dbms.Valid, nil
}

// Takes in a clause capturer and returns a different but semantically equivalent clause
func equivalenceTransform(impl translator.Implementation, clause *helperclauses.ClauseCapturer, seed *seed.Seed) {
	var subclausesAsClauses []translator.Clause
	// Transform the subclauses
	for _, subclause := range clause.GetSubclauseClauseCapturers() {
		equivalenceTransform(impl, subclause, seed)
		subclausesAsClauses = append(subclausesAsClauses, subclause)
	}

	// Transform the clause itself
	if transformer, ok := clause.GetCapturedClause().(translator.Transformer); ok && seed.BooleanWithProbability(0.25) {
		transformed := &TransformedClause{UseTransformed: true, origClause: clause.Copy()}
		if transformedClause := transformer.Transform(seed, clause.GetCapturedSchema(), subclausesAsClauses); transformedClause != nil {
			transformed.transformedClause = helperclauses.GetClauseCapturerForClause(transformedClause)
			clause.UpdateClause(transformed)
		}
	}
}

// Implements translator.Clause
//
// Represents a transformed clause.
// Which clause will be generated can be changed by setting useTransformed.
type TransformedClause struct {
	UseTransformed    bool
	origClause        *helperclauses.ClauseCapturer
	transformedClause *helperclauses.ClauseCapturer
}

func (c TransformedClause) getSelectedClause() *helperclauses.ClauseCapturer {
	if c.UseTransformed {
		return c.transformedClause
	}
	return c.origClause
}

// Generate subclauses for transformedClause
func (c *TransformedClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{c.getSelectedClause()}
}
