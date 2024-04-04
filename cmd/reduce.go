package cmd

import (
	"os"
	"strings"

	"github.com/Anon10214/dinkel/cmd/config"
	"github.com/Anon10214/dinkel/scheduler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// reduceCmd represents the reduce command
var reduceCmd = &cobra.Command{
	Use:   "reduce bugreport",
	Short: "Reduce a generated bugreport's queries",
	Args:  cobra.ExactArgs(1),
	Long: `Reduce a generated bugreport's queries.

The associated byte string has to generate the associated query.
This command then reduces the queries according to their strategy.`,
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
		conf.ByteString = bugreport.ByteString

		reducedReportPath := strings.ReplaceAll(args[0], ".yml", "_reduced")

		logrus.Infof("Reducing bugreport %s, reduced report will be stored at %s", args[0], reducedReportPath)

		// Store the bugreport report_x.yml at report_x_reduced.yml
		if err := scheduler.Reduce(conf, reducedReportPath); err != nil {
			logrus.Errorf("Reduction failed - %v", err)
		} else {
			logrus.Info("Reduction successful")
		}
	},
}

func init() {
	rootCmd.AddCommand(reduceCmd)
}
