package bisect

import (
	"fmt"
	"sync"

	"github.com/Anon10214/dinkel/cmd/config"
	"github.com/Anon10214/dinkel/cmd/rerun"
	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/biscepter/pkg/biscepter"
	"github.com/sirupsen/logrus"
)

func bisectBugreports(reports []*config.BugReport, bisectFix bool) error {
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

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(len(targets))

	for target, reports := range targets {
		if err := runJob(configMap[target], reports, waitGroup, bisectFix); err != nil {
			logrus.Fatalf("Failed to run bisection job for target %s - %v", target, err)
		}
	}

	// Wait for all bisections to be done
	waitGroup.Wait()
	return nil
}

func runJob(job *biscepter.Job, reports []*config.BugReport, wg *sync.WaitGroup, bisectFix bool) error {
	job.ReplicasCount = len(reports)
	rsChan, ocChan, err := job.Run()
	if err != nil {
		return err
	}

	go func(reports []*config.BugReport) {
		bisectionsLeft := len(reports)
		for bisectionsLeft != 0 {
			select {
			// Offending commit found
			case commit := <-ocChan:
				reports[commit.ReplicaIndex].OffendingCommit = commit.Commit
				fmt.Printf("Bisection done for target %s and report #%d! Offending commit: %s\nCommit message: %s\n", reports[0].Target, commit.ReplicaIndex, commit.Commit, commit.CommitMessage)

				if err := job.Stop(); err != nil {
					logrus.Errorf("Couldn't stop job - %v", err)
				}

				// Notify that we finished
				wg.Done()
				return

			// New system to test online
			case system := <-rsChan:
				// GDBMSs only have one port, we can assume this is always the right port
				report := reports[system.ReplicaIndex]

				conf, err := config.GetConfigForTarget(report.Target, targetConfigPath)
				if err != nil {
					logrus.Panicf("Couldn't read target from supplied bugreport: %v", err)
				}

				targetPort := system.Ports[job.Ports[0]]
				conf.DBOptions.Port = &targetPort

				conf.DBOptions.BackwardsCompatibleMode = true

				res, err := rerun.Rerun(report, conf)
				if err != nil && res != dbms.Crash {
					logrus.Panicf("Failed to rerun bugreport - %v", err)
				}

				logrus.Infof("Rerunning query resulted in return type %s", res.ToString())
				if res == dbms.Bug || res == dbms.Crash {
					logrus.Info("This system is bad!")
					if bisectFix {
						system.IsGood()
					} else {
						system.IsBad()
					}
				} else {
					logrus.Info("This system is good!")
					if bisectFix {
						system.IsBad()
					} else {
						system.IsGood()
					}
				}
			}
		}
	}(reports)

	return nil
}
