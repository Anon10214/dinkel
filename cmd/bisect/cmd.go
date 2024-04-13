package bisect

import (
	"bytes"
	"os"
	"path"
	"strings"

	"github.com/Anon10214/dinkel/cmd/config"
	"github.com/Anon10214/dinkel/biscepter/pkg/biscepter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var bisectConcurrency uint
var bisectFix bool
var targetConfigPath string
var targetConfigOverwrite map[string]string

var Cmd = &cobra.Command{
	Use:   "bisect",
	Short: "Bisects all bugreports in the given directory which have not been bisected yet",
	Long: `This command will bisect every bugreport present in the directory passed by the --bugreports flag using biscepter.
If the bugreport already has the offendingCommit field set, it will be ignored for the bisection.`,

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

		bugreportsDir, err := cmd.Flags().GetString("bugreports")
		if err != nil {
			logrus.Fatalf("Failed to get location of bugreports - %v", err)
		}

		// Not really a better way to do this, is needed to rerun the bug
		targetConfigPath, err = cmd.Flags().GetString("target-config")
		if err != nil {
			logrus.Fatalf("Couldn't get target config - %v", err)
		}

		reportsDir, err := os.ReadDir(bugreportsDir)
		if err != nil {
			logrus.Fatalf("Couldn't read in bug reports - %v", err)
		}

		var reports []*config.BugReport
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

		if err := bisectBugreports(reports, bisectFix); err != nil {
			logrus.Fatalf("Couldn't bisect bugreports - %v", err)
		}
		logrus.Info("Finished bisection!")
	},
}

func init() {
	Cmd.Flags().StringToStringVar(&targetConfigOverwrite, "config-overwrite", map[string]string{}, "Overwrite the biscepter-config of a target (target is either neo4j or redisgraph). Format: target_1=path.yml,...,target_n=path.yml")
	Cmd.Flags().UintVar(&bisectConcurrency, "max-concurrency", 0, "The max amount of replicas that can run concurrently, or 0 if no limit")
	Cmd.Flags().BoolVar(&bisectFix, "bisect-fix", false, "Bisect the the commit of a bugreport's fix instead of its root cause")
}
