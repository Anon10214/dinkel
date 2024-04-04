package rerun

import (
	"fmt"

	"github.com/Anon10214/dinkel/cmd/config"
	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/scheduler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "rerun bugreport",
	Short: "Rerun the query from a given bug report",
	Args:  cobra.ExactArgs(1),
	Long: `This command allows you to rerun a query from a bug report.

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

		res, err := Rerun(bugreport, conf)
		if err != nil && res != dbms.Crash {
			logrus.Fatalf("Rerunning bugreport didn't result in valid run - %v", err)
		}

		switch res {
		case dbms.Valid:
			logrus.Infof("Query was valid - %v", err)
		case dbms.Crash:
			logrus.Infof("Query caused database to crash - %v", err)
		default:
			logrus.Infof("statement resulted in non valid return type: %s", res.ToString())
		}
	},
}

func Rerun(bugreport *config.BugReport, conf scheduler.Config) (dbms.QueryResultType, error) {
	db := conf.DB

	logrus.Infof("Rerunning query for target %s", bugreport.Target)

	if err := db.Init(conf.DBOptions); err != nil {
		return 0, fmt.Errorf("failed to init DB - %v", err)
	}
	if err := db.Reset(conf.DBOptions); err != nil {
		return 0, fmt.Errorf("failed to reset DB - %v", err)
	}

	var queryResults []dbms.QueryResult

	// Rerun the query
	for i, statement := range bugreport.Query {
		// TODO: Find a better way to determine when to reset
		logrus.Infof("Rerunning statement #%d/%d", i, len(bugreport.Query))
		logrus.Debugf("Rerunning statement %s", statement)
		res := db.RunQuery(conf.DBOptions, statement)
		queryResults = append(queryResults, res)
		if ok, err := db.VerifyConnectivity(conf.DBOptions); !ok {
			return dbms.Crash, err
		}
		if resType := db.GetQueryResultType(res, conf.ErrorMessageRegex); resType != dbms.Valid {
			return resType, nil
		}
	}
	logrus.Infof("Done rerunning query")
	if res := bugreport.Strategy.ValidateRerunResults(queryResults, db); res != dbms.Valid {
		return res, nil
	}

	return dbms.Valid, nil
}
