/*
Package exporter provides a prometheus exporter for monitoring dinkel.

It provides metrics during fuzzing.
These metrics include:
  - query counts
  - generation latencies
  - query latencies
  - count of query result types

Additionally, there is the possibility of exposing "full" metrics, which are useful for benchmarking the fuzzer itself.
In full mode, in addition to the previous metrics, the fuzzer exposes the following data:
  - total size of all queries
  - total count of all keywords in all queries
  - query context data dependency count in all queries
  - AGS data dependency count in all queries

These metrics are exposed on the port passed to [RegisterExporter] on the /metrics endpoint.
*/
package prometheus

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/middleware"
	"github.com/Anon10214/dinkel/scheduler"
	"github.com/Anon10214/dinkel/scheduler/strategy"
	"github.com/Anon10214/dinkel/translator/helperclauses"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// dinkelExporter implements [middleware.Middleware].
type dinkelExporter struct {
	// Metrics for the fuzzing run
	queryCount          prometheus.Counter
	generationLatencies prometheus.Summary
	queryLatencies      prometheus.Summary

	// Metrics for query results
	queryResultCounters map[dbms.QueryResultType]prometheus.Counter

	// Fields used to calculate some metrics

	// Set by GetRootClause handler.
	// Used to calculate generation latencies.
	//  time(GetRootClause) - time(RunQuery) = generationLatency
	queryGenerationStart time.Time
}

// fullDinkelExporter embeds dinkelExporter,
// exposing additional metrics used for benchmarking the fuzzer
type fullDinkelExporter struct {
	// A semaphore making sure that not too many goroutines are analysing lastQueryClause at once
	// Currently hardcoded at 16 concurrent analyses
	analysisSemaphore semaphore.Weighted

	// All keywords considered for keyword metrics
	keywords []string

	lastQueryString string
	lastQueryClause *helperclauses.ClauseCapturer

	querySize prometheus.Counter

	totalKeywordCount prometheus.Counter
	keywordCount      map[string]prometheus.Counter

	qcDataDependencies  prometheus.Counter
	agsDataDependencies prometheus.Counter
}

// equivalenceTransformationDinkelExporter exports metrics for benchmarking the equivalence transformation strategy
type equivalenceTransformationDinkelExporter struct {
	// A semaphore making sure that not too many goroutines are analysing lastQueryClause at once
	// Currently hardcoded at 16 concurrent analyses
	analysisSemaphore semaphore.Weighted

	lastQueryClause         *helperclauses.ClauseCapturer
	lastQueryWasTransformed bool

	transformedQueries prometheus.Counter

	transformedClauses prometheus.Counter

	transformedQueriesResultCounters map[dbms.QueryResultType]prometheus.Counter

	addedQcDataDependencies  prometheus.Counter
	addedAgsDataDependencies prometheus.Counter
}

// RegisterExporter registers a new prometheus exporter for dinkel and exposes its metrics on the passed port.
//
// It returns a [scheduler.Config], where relevant fields are wrapped with middleware for collecting metrics.
func RegisterExporter(port int, conf *scheduler.Config, useFullExporter bool) {
	exporter := newExporter()

	middleware.RegisterMiddleware(exporter, conf)
	if useFullExporter {
		exporter := newFullExporter()
		middleware.RegisterMiddleware(exporter, conf)

		if conf.TargetStrategy == strategy.EquivalenceTransformation {
			exporter := newEquivalenceTransformationExporter()
			middleware.RegisterMiddleware(exporter, conf)
		}
	}

	// Expose metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		logrus.Infof("Listening on port %d, serving Prometheus metrics on /metrics", port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		logrus.Errorf("Prometheus endpoint terminated unexpectedly - %v", err)
	}()
}

// Returns a new exporter with initialized prometheus metrics
func newExporter() middleware.Middleware {
	return &dinkelExporter{
		queryCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "dinkel_query_count",
			Help: "Counter of the amount of queries generated",
		}),
		generationLatencies: promauto.NewSummary(prometheus.SummaryOpts{
			Name: "dinkel_generation_latency",
			Help: "Summary of query generation latencies",
		}),
		queryLatencies: promauto.NewSummary(prometheus.SummaryOpts{
			Name: "dinkel_query_latency",
			Help: "Summary of latencies encountered when sending the query to the target",
		}),

		queryResultCounters: map[dbms.QueryResultType]prometheus.Counter{
			dbms.Valid: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dinkel_valid_query_count",
				Help: "The amount of generated queries indicating valid queries",
			}),
			dbms.Invalid: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dinkel_invalid_query_count",
				Help: "The amount of generated queries indicating invalid queries",
			}),
			dbms.Bug: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dinkel_bug_query_count",
				Help: "The amount of generated queries indicating bugs",
			}),
			dbms.Crash: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dinkel_crash_query_count",
				Help: "The amount of generated queries which crashed the database",
			}),
			dbms.ReportedBug: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dinkel_reported_bug_query_count",
				Help: "The amount of generated queries indicating known bugs which have already been reported",
			}),
			dbms.Timeout: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dinkel_timeout_query_count",
				Help: "The amount of generated queries which triggered a timeout",
			}),
		},
	}
}

// Hooks returns the hooks for the prometheus exporter
func (e *dinkelExporter) Hooks() middleware.Hooks {
	return middleware.Hooks{
		StrategyHooks: getStrategyHooks(e),
		DBHooks:       getDBHooks(e),
	}
}

func newFullExporter() middleware.Middleware {
	exporter := fullDinkelExporter{
		analysisSemaphore: *semaphore.NewWeighted(16),

		keywords: []string{
			// Clauses
			"MATCH",
			"MERGE",
			"CREATE",
			"CALL",
			"UNWIND",
			"WITH",
			"DELETE",
			"REMOVE",
			"SET",
			"RETURN",
			"UNION",
			"FOREACH",

			// Expressions
			"COUNT",
			"EXISTS",
			"ALL",
			"CASE",

			// Literals
			"false",
			"true",
			"null",

			// Operators
			"AND",
			"NOT",
			"OR",
			"XOR",
		},

		totalKeywordCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "dinkel_total_keyword_count",
			Help: "The amount of times any keyword has appeared in queries",
		}),

		querySize: promauto.NewCounter(prometheus.CounterOpts{
			Name: "dinkel_query_size_sum",
			Help: "The total size of all queries generated",
		}),

		qcDataDependencies: promauto.NewCounter(prometheus.CounterOpts{
			Name: "dinkel_query_context_dependencies_count",
			Help: "How many data dependencies stemming from the query context have been generated",
		}),
		agsDataDependencies: promauto.NewCounter(prometheus.CounterOpts{
			Name: "dinkel_abstract_graph_summary_dependencies_count",
			Help: "How many data dependencies stemming from AGS have been generated",
		}),

		keywordCount: make(map[string]prometheus.Counter),
	}
	for _, keyword := range exporter.keywords {
		exporter.keywordCount[keyword] = promauto.NewCounter(prometheus.CounterOpts{
			Name: "dinkel_keyword_" + strings.ToLower(keyword) + "_count",
			Help: fmt.Sprintf("The amount of times the %q keyword has appeared in queries", keyword),
		})
	}

	return &exporter
}

// Hooks returns the hooks for the full prometheus exporter
func (e *fullDinkelExporter) Hooks() middleware.Hooks {
	return middleware.Hooks{
		StrategyHooks: getFullStrategyHooks(e),
		DBHooks:       getFullDBHooks(e),
	}
}

func newEquivalenceTransformationExporter() middleware.Middleware {
	exporter := equivalenceTransformationDinkelExporter{
		analysisSemaphore: *semaphore.NewWeighted(16),

		transformedQueries: promauto.NewCounter(prometheus.CounterOpts{
			Name: "dinkel_equivalence_transformed_queries_count",
			Help: "How many queries were transformed by the equivalence transformation strategy",
		}),

		transformedClauses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "dinkel_equivalence_transformed_clauses_count",
			Help: "How many clauses were transformed by the equivalence transformation strategy",
		}),

		addedQcDataDependencies: promauto.NewCounter(prometheus.CounterOpts{
			Name: "dinkel_equivalence_added_query_context_dependencies_count",
			Help: "How many data dependencies stemming from the query context have been added through transformations",
		}),
		addedAgsDataDependencies: promauto.NewCounter(prometheus.CounterOpts{
			Name: "dinkel_equivalence_added_abstract_graph_summary_dependencies_count",
			Help: "How many data dependencies stemming from AGS have been added through transformations",
		}),

		transformedQueriesResultCounters: map[dbms.QueryResultType]prometheus.Counter{
			dbms.Valid: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dinkel_equivalence_valid_query_count",
				Help: "The amount of transformed queries indicating valid queries",
			}),
			dbms.Invalid: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dinkel_equivalence_invalid_query_count",
				Help: "The amount of transformed queries indicating invalid queries",
			}),
			dbms.Bug: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dinkel_equivalence_bug_query_count",
				Help: "The amount of transformed queries indicating bugs",
			}),
			dbms.Crash: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dinkel_equivalence_crash_query_count",
				Help: "The amount of transformed queries which crashed the database",
			}),
			dbms.ReportedBug: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dinkel_equivalence_reported_bug_query_count",
				Help: "The amount of transformed queries indicating known bugs which have already been reported",
			}),
			dbms.Timeout: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dinkel_equivalence_timeout_query_count",
				Help: "The amount of transformed queries which triggered a timeout",
			}),
		},
	}

	return &exporter
}

// Hooks returns the hooks for the full prometheus exporter
func (e *equivalenceTransformationDinkelExporter) Hooks() middleware.Hooks {
	return middleware.Hooks{
		StrategyHooks: getEquivalenceTransformationStrategyHooks(e),
		DBHooks:       getEquivalenceTransformationDBHooks(e),
	}
}
