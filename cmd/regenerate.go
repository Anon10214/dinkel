package cmd

import (
	"os"
	"path"

	"github.com/Anon10214/dinkel/cmd/config"
	"github.com/Anon10214/dinkel/scheduler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var regenerateBugreport bool

var regenerateCmd = &cobra.Command{
	Use:     "regenerate bugreport",
	Aliases: []string{"regen"},
	Short:   "Regenerate a query from the byte string in a given bug report",
	Args:    cobra.ExactArgs(1),
	Long: `This command allows you to regenerate a query from a bug report.

By passing a generated bug report, this command reads in the associated byte string, uses it to regenerates the query and runs it.
This is useful for debugging dinkel, for rerunning just the query from a bugreport, check dinkel rerun.`,
	Run: func(cmd *cobra.Command, args []string) {
		bugreport, err := config.ReadBugreport(args[0])
		if err != nil {
			logrus.Errorf("Failed to get bugreport - %v", err)
			os.Exit(1)
		}

		conf, err := config.GetConfigForTarget(bugreport.Target, targetConfigPath)
		if err != nil {
			logrus.Errorf("Couldn't read target from supplied bugreport: %v", err)
			os.Exit(1)
		}
		conf.Strategy = bugreport.Strategy
		conf.TargetStrategy = bugreport.StrategyNum
		conf.QueryLimit = 1
		conf.ByteString = bugreport.ByteString
		conf.BugReportsDirectory, _ = path.Split(bugreport.FilePath)
		conf.SuppressBugreport = !regenerateBugreport
		conf.DisableKeybinds = true

		logrus.Infof("Regenerating query from %s for target %s", args[0], bugreport.Target)

		// Run the fuzzer
		if err := scheduler.Run(conf); err != nil {
			logrus.Errorf("Scheduler failed with: %v", err)
		} else {
			logrus.Infoln("Scheduler terminated without error")
		}
	},
}

func init() {
	rootCmd.AddCommand(regenerateCmd)

	regenerateCmd.Flags().BoolVarP(&regenerateBugreport, "regenerate-bugreport", "r", false, "Regenerate a bugreport if a bug is triggered")
}
