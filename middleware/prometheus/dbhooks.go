package prometheus

import (
	"time"

	"github.com/Anon10214/dinkel/dbms"
)

func getDBHooks(exporter *dinkelExporter) dbms.DBMiddleware {
	return dbms.DBMiddleware{
		RunQueryMiddleware:           exporter.handleRunQuery,
		VerifyConnectivityMiddleware: exporter.handleVerifyConnectivity,
	}
}

// handleRunQuery is the RunQuery handler for dinkelExporter.
// It measures the generation and query latency.
func (e *dinkelExporter) handleRunQuery(next dbms.RunQueryHandler) dbms.RunQueryHandler {
	generationLatency := time.Since(e.queryGenerationStart)
	e.generationLatencies.Observe(generationLatency.Seconds())

	return func(opts dbms.DBOptions, query string) dbms.QueryResult {
		startTime := time.Now()

		res := next(opts, query)

		latency := time.Since(startTime)
		e.queryLatencies.Observe(latency.Seconds())

		return res
	}
}

// handleVerifyConnectivity is the VerifyConnectivity handler for dinkelExporter.
// If the result of next indicates a crash, this handler increments the query result counter for crashes.
func (e *dinkelExporter) handleVerifyConnectivity(next dbms.VerifyConnectivityHandler) dbms.VerifyConnectivityHandler {
	return func(opts dbms.DBOptions) (bool, error) {
		ok, err := next(opts)

		if !ok {
			e.queryResultCounters[dbms.Crash].Inc()
		}

		return ok, err
	}
}

func getFullDBHooks(exporter *fullDinkelExporter) dbms.DBMiddleware {
	return dbms.DBMiddleware{
		RunQueryMiddleware: exporter.handleRunQuery,
	}
}

func (e *fullDinkelExporter) handleRunQuery(next dbms.RunQueryHandler) dbms.RunQueryHandler {
	return func(opts dbms.DBOptions, query string) dbms.QueryResult {
		e.lastQueryString = query
		res := next(opts, query)
		return res
	}
}
