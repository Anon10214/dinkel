package scheduler

import (
	"context"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
	"github.com/sirupsen/logrus"
)

// Reduce takes in a conf for regenerating the bugreport and a string
// holding the path where the reduced bugreport should be saved.
//
// It then attempts to reduce the bug-triggering query to a minimal, reproducible
// example triggering the bug. However, some further manual reduction will almost
// always be necessary afterwards.
func Reduce(conf Config, newBugreportName string) error {
	seed := seed.GetPregeneratedByteString(conf.ByteString)
	// Generate the original queries
	var origRootClauses []*helperclauses.ClauseCapturer
	var origResults []dbms.QueryResult

	conf.Strategy.Reset()

	if err := conf.DB.Init(conf.DBOptions); err != nil {
		return err
	}

	if err := conf.DB.Reset(conf.DBOptions); err != nil {
		return err
	}

	for {
		schema, err := conf.DB.GetSchema(conf.DBOptions)
		if err != nil {
			return err
		}

		rootClause := helperclauses.GetClauseCapturerForClause(conf.Strategy.GetRootClause(conf.Implementation, schema, seed), conf.Implementation)
		statement := translator.GenerateStatement(seed, schema, rootClause, conf.Implementation)
		origRootClauses = append(origRootClauses, rootClause)

		result := conf.DB.RunQuery(conf.DBOptions, statement)
		origResults = append(origResults, result)

		if conf.Strategy.DiscardQuery(conf.Strategy.GetQueryResultType(conf.DB, conf.DBOptions, result, conf.ErrorMessageRegex), conf.DB, conf.DBOptions, result, seed) {
			break
		}
	}

	logrus.Info("Reducing queries...")

	// Reduce the queries
	reductionContext := context.Background()
	reducedClauses := copyRootClauses(origRootClauses)
	for {
		copiedClauses := copyRootClauses(reducedClauses)
		ctx, newClauses, isDone := conf.Strategy.ReduceStep(reductionContext, copiedClauses)
		reductionContext = ctx

		results, err := getQueryResults(conf, newClauses)
		if err != nil {
			return err
		}

		if conf.Strategy.ValidateReductionResult(conf.DB, origResults, results) {
			logrus.Info("Reduction step successful")
			reducedClauses = newClauses
		} else {
			logrus.Info("Reduction step unsuccessful")
		}
		logrus.Tracef("Orig:\n%v\n\nNew:\n%v", origResults, results)

		if isDone {
			break
		}
	}

	logrus.Info("Reduction done")

	// Create bug report
	if err := conf.DB.Reset(conf.DBOptions); err != nil {
		return err
	}

	var query []string
	var lastResult dbms.QueryResult
	for _, rootClause := range reducedClauses {
		schema, err := conf.DB.GetSchema(conf.DBOptions)
		if err != nil {
			return err
		}
		statement := translator.GenerateStatement(seed, schema, rootClause, conf.Implementation)
		query = append(query, statement)

		lastResult = conf.DB.RunQuery(conf.DBOptions, statement)
	}
	writeBugReport(conf, lastResult, query, seed, newBugreportName)

	return nil
}

func copyRootClauses(orig []*helperclauses.ClauseCapturer) []*helperclauses.ClauseCapturer {
	var copiedRootClauses []*helperclauses.ClauseCapturer
	for _, clause := range orig {
		copiedRootClauses = append(copiedRootClauses, clause.Copy())
	}

	return copiedRootClauses
}

func getQueryResults(conf Config, rootClauses []*helperclauses.ClauseCapturer) ([]dbms.QueryResult, error) {
	seed := seed.GetPregeneratedByteString(conf.ByteString)

	if err := conf.DB.Reset(conf.DBOptions); err != nil {
		return nil, err
	}

	var queryResults []dbms.QueryResult
	for statements := 0; statements < len(rootClauses); statements++ {
		schema, err := conf.DB.GetSchema(conf.DBOptions)
		if err != nil {
			return nil, err
		}

		rootClause := rootClauses[statements]
		statement := translator.GenerateStatement(seed, schema, rootClause, conf.Implementation)

		result := conf.DB.RunQuery(conf.DBOptions, statement)
		queryResults = append(queryResults, result)
	}

	return queryResults, nil
}
