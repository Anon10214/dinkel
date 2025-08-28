package scheduler

import (
	"context"
	"errors"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/scheduler/strategy/equivalencetransformation"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
	"github.com/sirupsen/logrus"
)

// Reduce takes in a conf for regenerating the bugreport and a string
// holding the path where the reduced bugreport should be saved.
//
// If fullReduction is set to true, the reduction will be repeated until no more changes to any of the statements occur.
//
// It then attempts to reduce the bug-triggering query to a minimal, reproducible
// example triggering the bug. However, some further manual reduction will almost
// always be necessary afterwards.
func Reduce(conf Config, newBugreportName string, fullReduction bool) error {
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

	conf.Strategy.Reset()

	helperclauses.SetImplementation(conf.Implementation)
	for {
		schema, err := conf.DB.GetSchema(conf.DBOptions)
		if err != nil {
			return err
		}

		rootClause := helperclauses.GetClauseCapturerForClause(conf.Strategy.GetRootClause(conf.Implementation, schema, seed))
		statement, _ := translator.GenerateStatement(seed, schema, rootClause, conf.Implementation, 0)
		origRootClauses = append(origRootClauses, rootClause)

		result, err := RunQuery(conf, statement)
		if err != nil {
			return err
		}
		origResults = append(origResults, result)

		if result.Type == dbms.Crash {
			if ok, err := ConnectToDB(conf); !ok {
				return errors.Join(errors.New("couldn't recover database connection after crash"), err)
			}
		}

		if conf.Strategy.DiscardQuery(result.Type, conf.DB, conf.DBOptions, result, seed) {
			break
		}
	}

	logrus.Info("Reducing queries...")

	// Reduce the queries
	reductionContext := context.Background()
	reducedClauses := copyRootClauses(origRootClauses)
	// Track statements to see if they were further reduced, if full reduction is enabled
	var prevStatements []string
	var curStatements []string
	for {
		copiedClauses := copyRootClauses(reducedClauses)
		ctx, newClauses, isDone := conf.Strategy.ReduceStep(reductionContext, copiedClauses)
		reductionContext = ctx

		results, newStatements, err := getQueryResults(conf, newClauses)
		if err != nil {
			return err
		}
		curStatements = newStatements

		if conf.Strategy.ValidateReductionResult(conf.DB, origResults, results) {
			logrus.Info("Reduction step successful")
			reducedClauses = newClauses
		} else {
			logrus.Info("Reduction step unsuccessful")
		}
		logrus.Tracef("Orig:\n%v\n\nNew:\n%v", origResults, results)

		if isDone {
			if fullReduction && !isSameStatements(prevStatements, curStatements) {
				logrus.Infof("Reduction caused change in statements, repeating reduction")
				prevStatements = curStatements
				continue
			}
			break
		}
	}

	logrus.Info("Reduction done")
	logrus.Info("Creating bugreport")

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
		statement, _ := translator.GenerateStatement(seed, schema, rootClause, conf.Implementation, 0)
		query = append(query, statement)

		lastResult, err = RunQuery(conf, statement)
		if err != nil {
			return err
		}
		if lastResult.Type == dbms.Crash {
			if ok, err := ConnectToDB(conf); !ok {
				return errors.Join(errors.New("couldn't recover database connection after crash"), err)
			}
		}
	}
	WriteBugReport(conf, lastResult, query, "", seed, newBugreportName)

	return nil
}

func copyRootClauses(orig []*helperclauses.ClauseCapturer) []*helperclauses.ClauseCapturer {
	var copiedRootClauses []*helperclauses.ClauseCapturer
	for _, clause := range orig {
		copiedRootClauses = append(copiedRootClauses, clause.Copy())
	}

	return copiedRootClauses
}

func getQueryResults(conf Config, rootClauses []*helperclauses.ClauseCapturer) ([]dbms.QueryResult, []string, error) {
	var statements []string

	seed := seed.GetPregeneratedByteString(conf.ByteString)

	if err := conf.DB.Reset(conf.DBOptions); err != nil {
		return nil, nil, err
	}

	var queryResults []dbms.QueryResult
	for statementCount := 0; statementCount < len(rootClauses); statementCount++ {
		logrus.Debugf("Running statement #%d", statementCount)
		if _, ok := conf.Strategy.(*equivalencetransformation.Strategy); ok && statementCount == len(rootClauses)/2 {
			if err := conf.DB.Reset(conf.DBOptions); err != nil {
				return nil, nil, err
			}
		}
		schema, err := conf.DB.GetSchema(conf.DBOptions)
		if err != nil {
			return nil, nil, err
		}

		rootClause := rootClauses[statementCount]
		statement, _ := translator.GenerateStatement(seed, schema, rootClause, conf.Implementation, 0)

		statements = append(statements, statement)

		result, err := RunQuery(conf, statement)
		if err != nil {
			return nil, nil, err
		}
		if result.Type == dbms.Crash {
			if ok, err := ConnectToDB(conf); !ok {
				return nil, nil, errors.Join(errors.New("couldn't recover database connection after crash"), err)
			}
		}
		queryResults = append(queryResults, result)
	}

	return queryResults, statements, nil
}

func isSameStatements(prevStatements []string, curStatements []string) bool {
	if len(prevStatements) != len(curStatements) {
		return false
	}

	for i := 0; i < len(prevStatements); i++ {
		if prevStatements[i] != curStatements[i] {
			return false
		}
	}

	return true
}
