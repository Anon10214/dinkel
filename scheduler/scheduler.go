/*
Package scheduler holds the fuzzing scheduler, glueing together all parts needed for fuzzing.

This includes the target [dbms.DB], [translator.Implementation] and [strategy.FuzzingStrategy].
Additionally, the scheduler creates bug reports when a bug is found
and handles keybindings usable during fuzzing.

The scheduler is invoked through its [Run] function and can be configured using a [Config].
*/
package scheduler

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/scheduler/strategy"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Masterminds/sprig/v3"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/sirupsen/logrus"
)

// The Config for the scheduler
type Config struct {
	// The DB to run the queries against
	DB        dbms.DB
	DBOptions dbms.DBOptions
	// The OpenCypher implementation
	Implementation translator.Implementation
	// The fuzzing strategy for this run
	Strategy strategy.Strategy
	// The given byte string to use instead of generating a new one
	ByteString []byte
	// The seed to initialize the RNG to if no byte-string is given
	InitialSeed int64
	// How many nodes can be generated in the AST before aborting generation
	MaxASTNodes int64
	// How many times to retry connecting to the database before giving up
	DBConnectionRetries int
	// How long to wait before retrying to connect to the DB
	DBConnectionRetryInterval time.Duration
	// If true, not bug report will be created
	SuppressBugreport bool
	// Where bug reports should be written to
	BugReportsDirectory string
	// If true, the key bindings for the stats printer and adjusting the logging level won't be initialised
	DisableKeybinds bool
	// How many times to execute a fuzzing run by generating a query, -1 if unlimited
	QueryLimit int
	// The target DBMS. This only gets used for creating bug reports.
	TargetDB string
	// The target fuzzing strategy. This only gets used for creating bug reports.
	TargetStrategy strategy.FuzzingStrategy
	// ErrorMessageRegex holds regex strings, matching error messages the driver should ignore or treat as a previously reported bug.
	// These are read from a config in cmd/config/config.go.
	ErrorMessageRegex *dbms.ErrorMessageRegex
	// BugReportTemplate holds the template used to create the bugreport when a bug is found.
	BugReportTemplate *template.Template
}

// Stats of a fuzzing run
type fuzzingStats struct {
	sync.Mutex
	timestampStarted time.Time
	queries          int
	statements       int
	resultsByType    map[dbms.QueryResultType]int
}

// Run runs the fuzzer with the given config
func Run(conf Config) error {
	if ok, err := ConnectToDB(conf); !ok {
		return errors.Join(errors.New("failed to connect to database"), err)
	}

	stats := fuzzingStats{
		timestampStarted: time.Now(),
		resultsByType:    make(map[dbms.QueryResultType]int),
	}

	if !conf.DisableKeybinds {
		go initKeybinds(&stats)
	}

	for stats.queries = 0; conf.QueryLimit == -1 || stats.queries < conf.QueryLimit; stats.queries++ {
		var curSeed *seed.Seed
		if len(conf.ByteString) != 0 {
			curSeed = seed.GetPregeneratedByteString(conf.ByteString)
		} else {
			curSeed = seed.GetRandomByteString()
		}

		if err := conf.DB.Reset(conf.DBOptions); err != nil {
			return err
		}

		// The generated query
		var query []string

		conf.Strategy.Reset()

		// Generate statements
		for statementCount := 0; ; statementCount++ {
			schema, err := conf.DB.GetSchema(conf.DBOptions)
			if err != nil {
				return err
			}

			rootClause := conf.Strategy.GetRootClause(conf.Implementation, schema, curSeed)
			statement, err := translator.GenerateStatement(curSeed, schema, rootClause, conf.Implementation, conf.MaxASTNodes)
			if err != nil {
				logrus.Warnf("Generation failed, max AST nodes reached. Continuing with new query")
				break
			}
			query = append(query, statement)
			logrus.Debugf("Generated statement #%d:\n%s", statementCount, statement)

			res, err := RunQuery(conf, statement)
			if err != nil {
				return errors.Join(fmt.Errorf("couldn't run query %s", statement), err)
			}

			stats.Lock()
			stats.resultsByType[res.Type]++
			stats.statements++
			stats.Unlock()

			if res.Type == dbms.Timeout {
				break
			}

			if (res.Type == dbms.Bug || res.Type == dbms.Crash) && !conf.SuppressBugreport {
				query = conf.Strategy.PrepareQueryForBugreport(query)
				GenerateBugReport(conf, res, query, "", curSeed)
			}

			// Recover DBMS if this wasn't the last query
			if res.Type == dbms.Crash && statementCount != conf.QueryLimit-1 {
				logrus.Info("Trying to recover database connection after crash")
				if ok, err := ConnectToDB(conf); !ok {
					return errors.Join(errors.New("couldn't recover database connection after crash"), err)
				}
				logrus.Info("Database reinitialized")
				break
			}

			// Stop further generating query if DB decides it
			if conf.Strategy.DiscardQuery(res.Type, conf.DB, conf.DBOptions, res, curSeed) {
				break
			}
		}
	}

	if !conf.DisableKeybinds {
		printFuzzingStats(&stats, true)
	}

	return nil
}

func RunQuery(conf Config, query string) (dbms.QueryResult, error) {
	// Timeout the query manually after double the specified timeout
	// Ensures queries terminate even if the driver of GDBMS have a bug causing
	// them to run infinitely despite a specified timeout.
	timeoutChan := time.After(2 * conf.DBOptions.Timeout)
	resChan := make(chan dbms.QueryResult, 1)
	go func(c chan dbms.QueryResult) {
		c <- conf.DB.RunQuery(conf.DBOptions, query)
	}(resChan)

	var res dbms.QueryResult
	var forcedTimeOut bool
	select {
	case <-timeoutChan:
		forcedTimeOut = true
	case res = <-resChan:
	}

	if forcedTimeOut {
		// Kill query if driver or GDBMS o not respect given timeout
		logrus.Warnf("Had to kill query manually after it didn't terminate within double the specified timeout:\n%s", query)
		res.Type = dbms.Timeout
		return res, nil
	}

	if ok, _ := conf.DB.VerifyConnectivity(conf.DBOptions); !ok {
		logrus.Error("Query caused database to crash")
		res.Type = dbms.Crash
	} else {
		res.Type = conf.Strategy.GetQueryResultType(conf.DB, conf.DBOptions, res, conf.ErrorMessageRegex)
	}

	return res, nil
}

// Returns true if a connection to the DB has been established, else false.
// Uses options from the passed config to adjust behavior.
func ConnectToDB(conf Config) (bool, error) {
	var lastError error
	for i := 0; i <= conf.DBConnectionRetries || conf.DBConnectionRetries == -1; i++ {
		if err := conf.DB.Init(conf.DBOptions); err != nil {
			lastError = err
			logrus.Info("Couldn't establish DB connection, retrying in ", conf.DBConnectionRetryInterval.String())
			time.Sleep(conf.DBConnectionRetryInterval)
		} else {
			if ok, err := conf.DB.VerifyConnectivity(conf.DBOptions); !ok {
				lastError = err
				logrus.Info("Couldn't establish DB connection, retrying in ", conf.DBConnectionRetryInterval.String())
				time.Sleep(conf.DBConnectionRetryInterval)
			} else {
				logrus.Info("Successfully established DB connection")
				return true, nil
			}
		}
	}
	return false, lastError
}

// Writes the bug report to the default location
func GenerateBugReport(conf Config, res dbms.QueryResult, query []string, offendingCommit string, seed *seed.Seed) {
	WriteBugReport(conf, res, query, offendingCommit, seed, fmt.Sprintf("report_%d", time.Now().UnixMicro()))
}

// BugreportMarkdownData is the data passed to the bugreport template when writing a bugreport's markdown content
type BugreportMarkdownData struct {
	LastStatement    string           // The last statement that was run when the bug was triggered
	LastResult       dbms.QueryResult // The last result returned
	Statements       []string         // All the statements that were run
	StatementsString string           // All the statements that were run, joined with "\n---\n"
	Strategy         string           // The name of the strategy used
	OffendingCommit  string           // The hash of the bug-introducing commit

	LastResultType dbms.QueryResultType // The type of the last result
	IsValid        bool                 // If LastResultType is VALID
	IsInvalid      bool                 // If LastResultType is INVALID
	IsBug          bool                 // If LastResultType is BUG
	IsCrash        bool                 // If LastResultType is CRASH
	IsReportedBug  bool                 // If LastResultType is REPORTEDBUG
	IsTimeout      bool                 // If LastResultType is TIMEOUT
}

// checkBugReportDirectory checks if the bug reports directory exists and attempts to create it if it doesn't
func checkBugReportDirectory(conf Config) {
	if stat, err := os.Stat(conf.BugReportsDirectory); err != nil || !stat.IsDir() {
		logrus.Warnf("Bug reports directory does not exist, attempting to create it...")
		if err := os.Mkdir(conf.BugReportsDirectory, 0o755); err != nil {
			logrus.Errorf("Failed to create bug report directory - %v", err)
		} else {
			logrus.Infof("Created bug reports directory at %q", conf.BugReportsDirectory)
		}
	}
}

// WriteBugReport creates a bugreport with the passed name in the directory pointed to by the BugReportsDirectory specified in the passed [Config].
// If the current target has a template for bug report markdowns, this function writes the markdown as well.
func WriteBugReport(conf Config, res dbms.QueryResult, query []string, offendingCommit string, seed *seed.Seed, reportName string) {
	checkBugReportDirectory(conf)

	type bugReport struct {
		Target          string
		Strategy        strategy.FuzzingStrategy
		TimeFound       string
		OffendingCommit string
		ByteString      string
		ReportStatus    string
		Query           []string
	}
	newBugReport := bugReport{
		Target:          conf.TargetDB,
		Strategy:        conf.TargetStrategy,
		TimeFound:       time.Now().String(),
		OffendingCommit: offendingCommit,
		ByteString:      base64.StdEncoding.EncodeToString(seed.GetByteString()),
		ReportStatus:    "unconfirmed",
		Query:           make([]string, len(query)),
	}

	for i, el := range query {
		newBugReport.Query[i] = fmt.Sprintf("%q", el)
	}

	templateString := `target: {{ .Target }}
strategy: {{ .Strategy }}
# When dinkel found this bug
time_found: "{{ .TimeFound }}"
# The commit that introduced this bug
offending_commit: "{{ .OffendingCommit }}"
# The status of this bug report { unconfirmed | confirmed | fixed | rejected }
report_status: {{ .ReportStatus }}
# The byte string that generates a query triggering this bug
byte_string: "{{ .ByteString }}"
query: {{ range $index, $element := .Query }}
  - {{$element}}{{end}}
`

	tmpl, err := template.New("").Funcs(sprig.FuncMap()).Parse(templateString)
	if err != nil {
		logrus.Errorf("Failed to parse template string when generating bug report - %v: %v", err, newBugReport)
		return
	}

	filePath := path.Join(conf.BugReportsDirectory, reportName+".yml")
	file, err := os.Create(filePath)
	if err != nil {
		logrus.Errorf("Failed to create bug report file at %s - %v: %v", filePath, err, newBugReport)
		return
	}

	if err := tmpl.Execute(file, newBugReport); err != nil {
		logrus.Errorf("Failed to write bug report - %v: %v", err, newBugReport)
		return
	}
	logrus.Errorf("Bug found, created bug report %s", filePath)

	mdData := BugreportMarkdownData{
		LastResult:      res,
		Statements:      query,
		OffendingCommit: offendingCommit,
	}
	WriteBugReportMarkdown(conf, mdData, reportName)
}

// WriteBugReportMarkdown writes the markdown of the given bugreport data in the directory pointed to by the BugReportsDirectory specified in the passed [Config] with the given name.
// It only requires Statements, OffendingCommit, and LastResult to be set in the passed [BugreportMarkdownData].
// Strategy is read from the passed [Config].
func WriteBugReportMarkdown(conf Config, data BugreportMarkdownData, reportName string) {
	checkBugReportDirectory(conf)

	filePath := path.Join(conf.BugReportsDirectory, reportName+".md")
	mdFile, err := os.Create(filePath)
	if err != nil {
		logrus.Errorf("Failed to create bug report markdown file at %s - %v", filePath, err)
		return
	}

	// Set missing data
	data.LastStatement = data.Statements[len(data.Statements)-1]
	data.StatementsString = strings.Join(data.Statements, "\n---\n")
	data.Strategy = conf.TargetStrategy.ToString()

	data.LastResultType = data.LastResult.Type
	data.IsValid = data.LastResultType == dbms.Valid
	data.IsInvalid = data.LastResultType == dbms.Invalid
	data.IsBug = data.LastResultType == dbms.Bug
	data.IsCrash = data.LastResultType == dbms.Crash
	data.IsReportedBug = data.LastResultType == dbms.ReportedBug
	data.IsTimeout = data.LastResultType == dbms.Timeout

	if err := conf.BugReportTemplate.Execute(mdFile, data); err != nil {
		logrus.Errorf("Failed to write bug report markdown - %v", err)
		return
	}

	logrus.Infof("Created bug report markdown %s", filePath)
}

// Prints the stats of the current fuzzing run
func printFuzzingStats(stats *fuzzingStats, isPostRun bool) {
	if isPostRun {
		fmt.Printf("  %s %s %[1]s  \n\n", strings.Repeat("─", 20), fmt.Sprintf("Finished fuzzing, stats when run finished at %s", time.Now().Format("15:04:05")))
	} else {
		fmt.Printf("  %s %s %[1]s  \n\n", strings.Repeat("─", 42), fmt.Sprintf("Statistics at %s", time.Now().Format("15:04:05")))
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	title := table.Row{"Statements"}
	header := table.Row{""}                    // Name of the query result type
	count := table.Row{"encountered"}          // Count of the query result type encountered
	percentage := table.Row{"% of statements"} // Percentage of the query result type to total queries

	stats.Lock()
	defer stats.Unlock()

	// Iterate over possible query result types and add their stats
	for i := dbms.Valid; i <= dbms.Timeout; i++ {
		title = append(title, "Statements")
		header = append(header, i.ToString())
		count = append(count, stats.resultsByType[i])
		percentage = append(percentage, fmt.Sprintf("%#0.2f%%", 100*float64(stats.resultsByType[i])/float64(stats.statements)))
	}

	title = append(title, "Statements")
	header = append(header, "Total")
	count = append(count, stats.statements)

	t.AppendRow(title, table.RowConfig{AutoMerge: true})
	t.AppendSeparator()
	t.AppendRow(header)
	t.AppendSeparator()
	t.AppendRow(count)
	t.AppendRow(percentage)

	t.SetStyle(table.StyleRounded)
	t.Render()

	t = table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendRow(table.Row{"General Stats", "General Stats"}, table.RowConfig{AutoMerge: true})
	t.AppendSeparator()
	t.AppendRow(table.Row{"#queries", stats.queries})
	t.AppendSeparator()
	t.AppendRow(table.Row{"#statements / #queries", fmt.Sprintf("%.1f", float64(stats.statements)/float64(stats.queries))})
	t.AppendSeparator()
	t.AppendRow(table.Row{"time elapsed", time.Since(stats.timestampStarted).Round(time.Second)})

	t.SetStyle(table.StyleRounded)
	t.Render()
}

// Initialises the key bindings the user may use during fuzzing.
//
//	s: print fuzzing stats
//	v: decrease logging verbosity
//	V: increase logging verbosity
//
// Takes in a pointer to the current fuzzing stats.
func initKeybinds(stats *fuzzingStats) {
	if err := exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run(); err != nil {
		logrus.Warnf("Failed to initialise key bindings.\nDinkel will still be fuzzing the target, but you will be unable to see fuzzing stats or change the logger's verbosity on the fly.\nError: %v", err)
		return
	}
	// do not display entered characters on the screen
	if err := exec.Command("stty", "-F", "/dev/tty", "-echo").Run(); err != nil {
		logrus.Warnf("Failed to initialise key bindings.\nDinkel will still be fuzzing the target, but you will be unable to see fuzzing stats or change the logger's verbosity on the fly.\nError: %v", err)
		return
	}

	logrus.Infof("Initialised key bindings\n\tpress 's' to display fuzzing stats\n\tpress 'v' to decrease logging verbosity\n\tpress 'V' to increase logging verbosity")
	var b = make([]byte, 1)
	for {
		os.Stdin.Read(b)
		switch b[0] {
		case 's':
			printFuzzingStats(stats, false)
		case 'v':
			// Set logrus logging level to info so the user sees the logging info messages
			oldLevel := logrus.GetLevel()
			logrus.SetLevel(logrus.InfoLevel)
			// Don't decrease verbosity below error level
			if oldLevel != logrus.ErrorLevel {
				oldLevel--
				logrus.Printf("Decreased logging verbosity - level is now %s", oldLevel.String())
			} else {
				logrus.Printf("Could't further decrease logging verbosity, already at error level")
			}
			logrus.SetLevel(oldLevel)
		case 'V':
			// Set logrus logging level to info so the user sees the logging info messages
			oldLevel := logrus.GetLevel()
			logrus.SetLevel(logrus.InfoLevel)
			// Don't increase verbosity above trace level
			if oldLevel != logrus.TraceLevel {
				oldLevel++
				logrus.Printf("Increased logging verbosity - level is now %s", oldLevel.String())
			} else {
				logrus.Println("Could't further increase logging verbosity, already at trace level")
			}
			logrus.SetLevel(oldLevel)
		}
	}
}
