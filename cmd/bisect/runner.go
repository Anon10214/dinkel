package bisect

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"

	"github.com/CelineWuest/biscepter/pkg/biscepter"
	"github.com/Anon10214/dinkel/cmd/config"
	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/scheduler"
	"github.com/Anon10214/dinkel/seed"
	"github.com/sirupsen/logrus"
)

func bisectBugreports(reports []*config.BugReport, invertBisection bool) error {
	targets := make(map[string][]*config.BugReport)

	for _, report := range reports {
		targets[report.Target] = append(targets[report.Target], report)
		// Make sure it is a valid target for bisection
		if _, found := configMap[report.Target]; !found {
			return fmt.Errorf("target %q cannot be bisected", report.Target)
		}
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.Debug("Target map:")
		for target, reports := range targets {
			logrus.Debugf("\t- %s: %d", target, len(reports))
		}
	}

	// Handle interrupts
	jobDoneChan := make(chan struct{})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		select {
		case <-ctx.Done():
			logrus.Infof("Captured an interrupt signal, commencing graceful shutdown of job. Interrupt again to force shutdown.")
			stop()
			gracefulShutdown(targets)
		case <-jobDoneChan:
		}
	}()

	// Handle panics
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Captured a panic: %v", r)
			logrus.Errorf("Stack trace: %s", debug.Stack())
			logrus.Infof("Attempting to gracefully shut down jobs")
			gracefulShutdown(targets)
		}
	}()

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(len(targets))

	for target, reports := range targets {
		job := configMap[target]
		job.MaxConcurrentReplicas = bisectConcurrency
		if err := runJob(job, reports, waitGroup, invertBisection); err != nil {
			logrus.Fatalf("Failed to run bisection job for target %s - %v", target, err)
		}
	}

	// Wait for all bisections to be done
	waitGroup.Wait()
	jobDoneChan <- struct{}{}
	logrus.Info("Finished all bisections!")
	return nil
}

func gracefulShutdown(targets map[string][]*config.BugReport) {
	for target := range targets {
		job := configMap[target]
		if err := job.Stop(); err != nil {
			logrus.Errorf("Failed to gracefully shut down job - %v", err)
		} else {
			logrus.Infof("Gracefully shut down job")
		}
	}
	os.Exit(1)
}

func runJob(job *biscepter.Job, reports []*config.BugReport, wg *sync.WaitGroup, invertBisection bool) error {
	logrus.Infof("Spinning up commit %s to get bad results to compare to", configMap[reports[0].Target].BadCommit)
	rs, err := job.RunCommitByHash(configMap[reports[0].Target].BadCommit)
	if err != nil {
		return err
	}

	conf, err := config.GetConfigForTarget(reports[0].Target, targetConfigPath)
	if err != nil {
		logrus.Panicf("Couldn't read target for target %s: %v", reports[0].Target, err)
	}
	rsPort := rs.Ports[job.Ports[0]]
	conf.DBOptions.Port = &rsPort
	conf.DBOptions.BackwardsCompatibleMode = true

	origResults := [][]dbms.QueryResult{}
	for i, report := range reports {
		conf.Strategy = report.Strategy

		logrus.Infof("Getting original result for report #%d/%d (%s)", i+1, len(reports), report.ReportName)

		results, err := getQueryResults(*report, conf, 0)
		if err != nil {
			return errors.Join(fmt.Errorf("couldn't get query results for report %s", report.ReportName), err)
		}

		origResults = append(origResults, results)
	}

	// Terminate the running system
	rs.Done()

	logrus.Infof("Finished gathering original results")

	job.ReplicasCount = len(reports)
	rsChan, ocChan, err := job.Run()
	if err != nil {
		return err
	}

	go func(reports []*config.BugReport, origResults [][]dbms.QueryResult) {
		bisectionsLeft := len(reports)
		lastRes := make([]dbms.QueryResult, len(reports))
		for bisectionsLeft != 0 {
			select {
			// Offending commit found
			case commit := <-ocChan:
				reports[commit.ReplicaIndex].OffendingCommit = commit.Commit
				logrus.Infof("Bisection done for target %s and report #%d! Offending commit: %s\nCommit message: %s\nPossible other commits: %v", reports[0].Target, commit.ReplicaIndex, commit.Commit, commit.CommitMessage, commit.PossibleOtherCommits)

				conf, err := config.GetConfigForTarget(reports[0].Target, targetConfigPath)
				if err != nil {
					logrus.Panicf("Couldn't read target from supplied bugreport: %v", err)
				}

				conf.TargetStrategy = reports[commit.ReplicaIndex].StrategyNum

				// Write new bugreport
				scheduler.WriteBugReport(
					conf,
					lastRes[commit.ReplicaIndex],
					reports[commit.ReplicaIndex].Query,
					commit.Commit,
					seed.GetPregeneratedByteString(reports[commit.ReplicaIndex].ByteString),
					reports[commit.ReplicaIndex].ReportName+"_bisected",
				)

				bisectionsLeft--
			// New system to test online
			case system := <-rsChan:
				logrus.Infof("Got a system for index %d", system.ReplicaIndex)
				report := reports[system.ReplicaIndex]

				conf, err := config.GetConfigForTarget(report.Target, targetConfigPath)
				if err != nil {
					logrus.Panicf("Couldn't read target from supplied bugreport: %v", err)
				}

				// GDBMSs only have one port, we can assume this is always the right port
				targetPort := system.Ports[job.Ports[0]]
				conf.DBOptions.Port = &targetPort
				conf.DBOptions.BackwardsCompatibleMode = true
				conf.Strategy = report.Strategy

				results, err := getQueryResults(*report, conf, system.ReplicaIndex)
				if err != nil {
					logrus.Panicf("Failed to get query results - %v", err)
				}

				logrus.Infof("Finished getting query results for index %d", system.ReplicaIndex)

				// If the result hasn't changed, the bug is still present
				if conf.Strategy.ValidateReductionResult(conf.DB, results, origResults[system.ReplicaIndex]) {
					logrus.Info("This system is bad!")
					if invertBisection {
						system.IsGood()
					} else {
						system.IsBad()
					}
				} else {
					logrus.Info("This system is good!")
					if invertBisection {
						system.IsBad()
					} else {
						system.IsGood()
					}
				}
			}
		}

		if err := job.Stop(); err != nil {
			logrus.Errorf("Couldn't stop job - %v", err)
		}

		// Notify that we finished
		wg.Done()
	}(reports, origResults)

	return nil
}

// getQueryResults runs the passed [config.BugReport] on the given [scheduler.Config] and returns the query results
// each statement of the report generated.
func getQueryResults(report config.BugReport, conf scheduler.Config, replicaIndex int) ([]dbms.QueryResult, error) {

	if ok, err := scheduler.ConnectToDB(conf); !ok {
		return nil, fmt.Errorf("failed to connect to DB - %v", err)
	}
	if err := conf.DB.Reset(conf.DBOptions); err != nil {
		return nil, fmt.Errorf("failed to reset DB - %v", err)
	}

	results := []dbms.QueryResult{}
	statementIndex := 0

	_, err := conf.Strategy.RerunQuery(report.Query, conf.DB, conf.DBOptions, func() (dbms.QueryResult, error) {
		statement := report.Query[statementIndex]
		statementIndex++
		logrus.Infof("Rerunning statement #%d/%d for index %d", statementIndex, len(report.Query), replicaIndex)
		logrus.Debugf("Rerunning statement %s for index %d", statement, replicaIndex)
		res, err := scheduler.RunQuery(conf, statement)
		results = append(results, res)
		return res, err
	})
	if err != nil {
		return nil, err
	}

	// Add empty query results for statements not executed - implying early crash or invalid query
	for i := len(results); i < len(report.Query); i++ {
		results = append(results, dbms.QueryResult{})
	}

	return results, nil
}
