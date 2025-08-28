package bisect

import (
	"bytes"
	"os"
	"path"
	"strings"

	"github.com/CelineWuest/biscepter/pkg/biscepter"
	"github.com/Anon10214/dinkel/cmd/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var bisectConcurrency uint
var invertBisection bool
var targetConfigPath string
var targetConfigOverwrite map[string]string

var Cmd = &cobra.Command{
	Use:   "bisect [bugreport...]",
	Short: "Bisects all bugreports in the given directory which have not been bisected yet, or optionally the reports passed to the command as arguments.",
	Long: `This command will bisect every bugreport present in the directory passed by the --bugreports flag using biscepter.
If bugreports are passed explicitly to this command as arguments, only these report will be bisected, regardless of whether they already have an offending commit assigned.
If the bugreport already has the offendingCommit field set, it will be ignored for the bisection.

Keep in mind that this bisection bisects the database based on the query result on the bad commit (i.e. the newest one).
This means that if your query no longer triggers a bug in the newest version, this command will find the latest commit where the bug is present instead.
In other words, it will not bisect the root cause of the bug, but rather the commit that fixes it.
If this is undesired, run this command with the --invert flag set.`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Validating target config overwrite
		for target := range targetConfigOverwrite {
			target = strings.ToLower(target)
			if target != "neo4j" &&
				target != "redisgraph" {
				logrus.Fatalf("Unknown target %q, must be one of {neo4j, redisgraph}", target)
			}
		}

		for target, conf := range targetConfigOverwrite {
			conf, err := os.ReadFile(conf)
			if err != nil {
				logrus.Fatalf("Failed to read config of target %q - %v", target, err)
			}
			configMap[target], err = biscepter.GetJobFromConfig(bytes.NewReader(conf))
			if err != nil {
				logrus.Fatalf("Couldn't parse job config from target %q - %v", target, err)
			}
			configMap[target].Log = logrus.StandardLogger()
			logrus.Infof("Updated config of target %q", target)
		}

		// Not really a better way to do this, is needed to rerun the bug
		var err error
		targetConfigPath, err = cmd.Flags().GetString("target-config")
		if err != nil {
			logrus.Fatalf("Couldn't get target config - %v", err)
		}

		var reports []*config.BugReport

		if len(args) == 0 {
			// If no args passed, bisect all unbisected reports in bugreports directory

			bugreportsDir, err := cmd.Flags().GetString("bugreports")
			if err != nil {
				logrus.Fatalf("Failed to get location of bugreports - %v", err)
			}

			reportsDir, err := os.ReadDir(bugreportsDir)
			if err != nil {
				logrus.Fatalf("Couldn't read in bug reports - %v", err)
			}

			for _, report := range reportsDir {
				if report.Type().IsRegular() {
					if strings.HasSuffix(report.Name(), ".yml") {
						bugreport, err := config.ReadBugreport(path.Join(bugreportsDir, report.Name()))
						if err != nil {
							logrus.Fatalf("Couldn't read in bug report %s - %v", report.Name(), err)
						}

						// Only bisect if offending commit not set
						if bugreport.OffendingCommit == "" {
							logrus.Debugf("Adding report %s", report.Name())
							reports = append(reports, bugreport)
						}
					}
				}
			}

			logrus.Infof("Found %d bugreports which have not been bisected yet.", len(reports))
		} else {
			// If args passed, treat them as bugreports and bisect them

			for _, report := range args {
				if !strings.HasSuffix(report, ".yml") {
					logrus.Fatalf("Report %s is not a .yml file", report)
				}

				bugreport, err := config.ReadBugreport(report)
				if err != nil {
					logrus.Fatalf("Couldn't read in bug report %s - %v", report, err)
				}

				if bugreport.OffendingCommit != "" {
					logrus.Warnf("Report %s already was already bisected, bisecting it again", report)
				}

				logrus.Debugf("Adding report %s", report)
				reports = append(reports, bugreport)
			}
		}

		if err := bisectBugreports(reports, invertBisection); err != nil {
			logrus.Fatalf("Couldn't bisect bugreports - %v", err)
		}
		logrus.Info("Finished bisection!")
	},
}

func init() {
	Cmd.Flags().StringToStringVar(&targetConfigOverwrite, "config-overwrite", map[string]string{}, "Overwrite the biscepter-config of a target (target is either neo4j or redisgraph). Format: target_1=path.yml,...,target_n=path.yml")
	Cmd.Flags().UintVar(&bisectConcurrency, "max-concurrency", 0, "The max amount of replicas that can run concurrently, or 0 if no limit")
	Cmd.Flags().BoolVar(&invertBisection, "invert", false, "Invert the direction into which to bisect, i.e. find the newest, instead of oldest, commit where the result differs from the most recent version")
}
