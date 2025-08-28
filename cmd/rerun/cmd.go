package rerun

import (
	"fmt"
	"path"

	"github.com/Anon10214/dinkel/cmd/config"
	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/scheduler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var RegenerateMarkdown bool

var Cmd = &cobra.Command{
	Use:   "rerun bugreport",
	Short: "Rerun the query from a given bug report",
	Args:  cobra.ExactArgs(1),
	Long: `This command allows you to rerun a query from a bug report.

By using the -r flag, you may also regenerate the markdown of the passed bug report.

By passing a generated bug report, this command reads in the included query and runs it against the target database.
This is useful for ensuring a bug's validity, for example if a new version of the target got released.
For debugging dinkel by regenerating a query given a bug report's byte string, check dinkel regenerate.`,
	Run: func(cmd *cobra.Command, args []string) {
		bugreport, err := config.ReadBugreport(args[0])
		if err != nil {
			logrus.Fatalf("Failed to get bugreport - %v", err)
		}

		targetConfig, err := cmd.Flags().GetString("target-config")
		if err != nil {
			logrus.Fatalf("Couldn't get target config - %v", err)
		}

		conf, err := config.GetConfigForTarget(bugreport.Target, targetConfig)
		if err != nil {
			logrus.Fatalf("Couldn't read target from supplied bugreport: %v", err)
		}

		conf.Strategy = bugreport.Strategy
		conf.TargetStrategy = bugreport.StrategyNum
		conf.BugReportsDirectory, _ = path.Split(bugreport.FilePath)

		res, err := Rerun(bugreport, conf)
		if err != nil && res.Type != dbms.Crash {
			logrus.Fatalf("Rerunning bugreport didn't result in valid run - %v", err)
		}

		switch res.Type {
		case dbms.Valid:
			logrus.Infof("Query was valid - %v", err)
		case dbms.Crash:
			logrus.Infof("Query caused database to crash - %v", err)
		default:
			logrus.Infof("statement resulted in non valid return type: %s", res.Type.ToString())
		}

		if RegenerateMarkdown {
			logrus.Infof("Regenerating report markdown")
			data := scheduler.BugreportMarkdownData{
				Statements:      bugreport.Query,
				LastResult:      res,
				OffendingCommit: bugreport.OffendingCommit,
			}
			scheduler.WriteBugReportMarkdown(conf, data, bugreport.ReportName)
		}
	},
}

func Rerun(bugreport *config.BugReport, conf scheduler.Config) (dbms.QueryResult, error) {
	db := conf.DB

	logrus.Infof("Rerunning query for target %s", bugreport.Target)

	if err := db.Init(conf.DBOptions); err != nil {
		return dbms.QueryResult{}, fmt.Errorf("failed to init DB - %v", err)
	}
	if err := db.Reset(conf.DBOptions); err != nil {
		return dbms.QueryResult{}, fmt.Errorf("failed to reset DB - %v", err)
	}

	// Rerun the query
	var lastRes dbms.QueryResult
	statementIndex := 0
	res, err := bugreport.Strategy.RerunQuery(bugreport.Query, conf.DB, conf.DBOptions, func() (dbms.QueryResult, error) {
		statement := bugreport.Query[statementIndex]
		statementIndex++
		logrus.Infof("Rerunning statement #%d/%d", statementIndex, len(bugreport.Query))
		logrus.Debugf("Rerunning statement %s", statement)
		res, err := scheduler.RunQuery(conf, statement)
		lastRes = res
		return res, err
	})

	if err != nil {
		return lastRes, err
	}
	logrus.Infof("Done rerunning query")

	lastRes.Type = res
	return lastRes, nil
}
