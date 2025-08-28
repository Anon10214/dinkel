// Package cmd provides the basic cobra-cli commands for fuzzing.
package cmd

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/Anon10214/dinkel/cmd/config"
	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/scheduler"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var rootCmd = &cobra.Command{
	Use:   "dinkel",
	Short: "A GDBMS fuzzer",
	Long:  `Dinkel is a fuzzer targeting GDBMSs, written entirely in Go.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Set the dbPort flag
		if cmd.Flags().Changed("port") {
			port, err := cmd.Flags().GetInt("port")
			if err != nil {
				logrus.Errorf("Failed to set DB port %v", err)
				os.Exit(1)
			}
			dbPort = &port
		}

		config.SetDefaultConfig(scheduler.Config{
			DBOptions: dbms.DBOptions{
				Host:    connectionString,
				Port:    dbPort,
				Timeout: time.Duration(dbTimeoutSeconds * float64(time.Second)),
			},
			BugReportsDirectory:       bugreportsDirectory,
			DBConnectionRetries:       dbConnectionRetries,
			DBConnectionRetryInterval: time.Duration(dbConnectionRetryInterval) * time.Second,
		})

		// Set the loggers verbosity
		if verbosity <= 0 {
			logrus.SetLevel(logrus.ErrorLevel)
		} else if verbosity <= 1 {
			logrus.SetLevel(logrus.InfoLevel)
		} else if verbosity <= 2 {
			logrus.SetLevel(logrus.DebugLevel)
		} else {
			logrus.SetLevel(logrus.TraceLevel)
		}

		// Set logger output
		if silenceTty || logFile != "" {
			var outputs []io.Writer
			if !silenceTty {
				outputs = append(outputs, os.Stderr)
			}
			if logFile != "" {
				file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
				if err != nil {
					logrus.Errorf("Failed to open or create log file at %s - %v", logFile, err)
					os.Exit(1)
				}
				if _, err := file.WriteString(strings.Repeat("-", 30) + " Start of new log entry " + strings.Repeat("-", 30) + "\n"); err != nil {
					logrus.Errorf("Failed to write to log file at %s - %v", logFile, err)
					os.Exit(1)
				}
				outputs = append(outputs, file)
			}
			logrus.SetOutput(io.MultiWriter(outputs...))
		}
	},
}

var verbosity int
var silenceTty bool
var logFile string

var dbTimeoutSeconds float64

var dbConnectionRetryInterval int
var dbConnectionRetries int

var connectionString string
var dbPort *int // Gets set in pre run
var dbPortFlag int

var bugreportsDirectory string

var targetConfigPath string

// Execute the cobra-cli root command
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Hide the completion command
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	// Init the logger
	formatter := prefixed.TextFormatter{
		TimestampFormat: "15:04:05.000",
		FullTimestamp:   true,
		ForceFormatting: true,
	}
	formatter.SetColorScheme(&prefixed.ColorScheme{
		TimestampStyle: "245",
	})
	logrus.SetFormatter(&formatter)

	rootCmd.PersistentFlags().IntVarP(&verbosity, "verbose", "v", 1, "Set the verbosity [0-3]")
	rootCmd.PersistentFlags().BoolVarP(&silenceTty, "silence-tty", "s", false, "Suppress the TTY logging output")
	rootCmd.PersistentFlags().StringVarP(&logFile, "log-file", "l", "", "File the log will be written to (Using this flag disables colors in the tty)")
	rootCmd.PersistentFlags().StringVar(&connectionString, "db-host", "localhost", "The database host to connect to")
	rootCmd.PersistentFlags().IntVar(&dbPortFlag, "port", 0, "The database port to connect to (Default decided by target DB driver)")
	rootCmd.PersistentFlags().Float64VarP(&dbTimeoutSeconds, "timeout", "t", 15, "How long until DB requests are considered timed out in seconds")
	rootCmd.PersistentFlags().IntVar(&dbConnectionRetries, "db-connection-retries", 3, "How many times to retry connecting to DB before giving up, or -1 if infinite")
	rootCmd.PersistentFlags().IntVar(&dbConnectionRetryInterval, "db-connection-interval", 15, "How many seconds to wait before retrying to connect to DB")
	rootCmd.PersistentFlags().StringVar(&bugreportsDirectory, "bugreports", "bugreports", "Where bugreports are to be stored")
	rootCmd.PersistentFlags().StringVarP(&targetConfigPath, "target-config", "c", "targets-config.yml", "The path to the target config")
}
