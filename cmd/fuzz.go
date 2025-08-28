package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Anon10214/dinkel/cmd/config"
	"github.com/Anon10214/dinkel/middleware/prometheus"
	"github.com/Anon10214/dinkel/scheduler"
	"github.com/Anon10214/dinkel/scheduler/strategy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var queryLimit int
var maxASTNodes int64
var disableKeybinds bool

var prometheusPort int
var prometheusFullMetrics bool

var fuzzCmd = &cobra.Command{
	Use:   "fuzz target [strategy]",
	Short: "Fuzz a target",
	Long: `Fuzz a provided target using an optional strategy.

Valid targets are:
    neo4j          - default port: 7687
    falkodb        - default port: 6379
    apache-age     - default port: 5432
    memgraph       - default port: 7687
    redisgraph     - default port: 6379 (DEPRECATED)

Valid strategies are:
    0 | NONE             (default)  - Generate random queries, hoping to trigger exceptions or crashes.
    1 | EQUIVALENCE_TRANSFORMATION  - Generate a random query and verify its result matches the one
                                      produced by the same query after it was transformed to a semantically
                                      equivalent.
    2 | PREDICATE_PARTITIONING      - First, generate write clauses to populate the schema.
                                      Then, generate a read query (1).
                                      Afterwards, generate a predicate Q. Verify, that the result
                                      of the queries with the predicates Q, NOT Q and Q IS NULL 
                                      produce the same results as the original query (1).`,
	Args: cobra.MatchAll(cobra.MinimumNArgs(1), cobra.MaximumNArgs(2), cobra.OnlyValidArgs),
	ValidArgs: []string{
		"neo4j", "redisgraph", "falkordb", "memgraph", "apache-age",
		"0", "NONE", "none",
		"1", "EQUIVALENCE_TRANSFORMATION", "equivalence_transformation",
		"2", "PREDICATE_PARTITIONING", "predicate_partitioning",
	},
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := config.GetConfigForTarget(args[0], targetConfigPath)
		if err != nil {
			fmt.Printf("Failed to initialize fuzzer - %v\n\n%s", err, cmd.Long)
			os.Exit(1)
		}

		conf.TargetStrategy = strategy.None
		// If strategy supplied
		if len(args) == 2 {
			switch strings.ToLower(args[1]) {
			case "0", "none":
				break
			case "1", "equivalence_transformation":
				conf.TargetStrategy = strategy.EquivalenceTransformation
			case "2", "predicate_partitioning":
				conf.TargetStrategy = strategy.PredicatePartitioning
			default:
				fmt.Printf("Failed to initialize fuzzer - invalid strategy\n\n%s", cmd.Long)
				os.Exit(1)
			}
		}
		conf.Strategy = conf.TargetStrategy.ToStrategy()
		conf.QueryLimit = queryLimit
		conf.MaxASTNodes = maxASTNodes
		conf.DisableKeybinds = disableKeybinds

		logrus.Infof("Starting up fuzzer for target %s", args[0])

		// Register Prometheus exporter if flag set
		if cmd.Flags().Changed("prometheus-port") {
			// Use returned fuzzing config
			prometheus.RegisterExporter(prometheusPort, &conf, prometheusFullMetrics)
		}

		// Run the fuzzer
		if err := scheduler.Run(conf); err != nil {
			logrus.Errorf("Scheduler failed with: %v", err)
		} else {
			logrus.Infoln("Scheduler terminated without error")
		}
	},
}

func init() {
	rootCmd.AddCommand(fuzzCmd)

	fuzzCmd.Flags().IntVarP(&queryLimit, "query-limit", "q", -1, "How many queries to generate before terminating. -1 if infinite")
	fuzzCmd.Flags().Int64Var(&maxASTNodes, "max-ast-nodes", 10_000, "How many AST nodes should be generated at most for a single statement. <= 0 if no limit")
	fuzzCmd.Flags().BoolVar(&disableKeybinds, "disable-keybinds", false, "If set, key bindings for the stats printer and adjusting logging won't be initialized")
	rootCmd.PersistentFlags().IntVar(&prometheusPort, "prometheus-port", 0, "Activate the prometheus exporter and set the port where Prometheus listens for requests on the /metrics endpoint")
	rootCmd.PersistentFlags().BoolVar(&prometheusFullMetrics, "prometheus-full-metrics", false, "Expose full prometheus metrics.\nThese are mostly just useful for benchmarking the fuzzer and don't provide a lot of value if the goal is to just test a target.")
}
