//go:build integration

package integration_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/Anon10214/dinkel/cmd/config"
	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/scheduler"
	"github.com/stretchr/testify/assert"
)

// Options for connection establishment
const (
	connectionRetries       int           = 15
	connectionRetryInterval time.Duration = time.Second
)

func TestIntegrationModels(t *testing.T) {
	for _, model := range []struct {
		name string
		port int
	}{
		{"neo4j", 1000},
		{"memgraph", 1001},
		{"falkordb", 1002},
	} {
		model := model // Capture model in loop

		t.Run("Test "+model.name, func(t *testing.T) {
			// Can target different models in parallel
			t.Parallel()

			dbOptions := dbms.DBOptions{
				// Containers expose ports to localhost
				Host: "localhost",
				// Always use default DBMS port
				Port: &model.port,
				// No big queries, 5 seconds suffices
				Timeout: 5 * time.Second,
			}

			conf, err := config.GetConfigForTarget(model.name, "../targets-config.yml")
			if assert.NoError(t, err, "Getting config for target failed") {
				assert.NoError(t, conf.BugReportTemplate.Execute(&bytes.Buffer{}, scheduler.BugreportMarkdownData{}), "Failed to execute bug report template")

				driver := conf.DB

				if assert.NoError(t, driver.Init(dbOptions), "Failed to init DB") {

					// Attempt to establish connection
					var connectionEstablished bool
					for i := 0; i < connectionRetries; i++ {
						if ok, _ := driver.VerifyConnectivity(dbOptions); ok {
							connectionEstablished = true
							break
						}
						t.Logf("Couldn't establish connection after %d/%d tries, retrying in %s", i+1, connectionRetries, connectionRetryInterval)
						time.Sleep(connectionRetryInterval)
					}

					// Fail if connection couldn't be established
					if !assert.True(t, connectionEstablished, "Failed to connect to DB") {
						t.FailNow()
					}
				}

				t.Run("GetSchema works", func(t *testing.T) {
					t.Run("Node labels fetched correctly", func(t *testing.T) {
						if !assert.NoError(t, driver.Reset(dbOptions), "Failed to reset DB") {
							return
						}

						label := "NODE_LABEL"

						// Check that labels get read out of the DB
						res := driver.RunQuery(dbOptions, fmt.Sprintf("CREATE (:%s)", label))
						assert.Equal(t, driver.GetQueryResultType(res, conf.ErrorMessageRegex), dbms.Valid, "Simple query causes non-valid result type")

						s, err := driver.GetSchema(dbOptions)
						if !assert.NoError(t, err, "Failed to get schema") {
							return
						}

						assert.Len(t, s.Labels[schema.ANY], 1, "More or less than one label fetched for ANY type")
						assert.Len(t, s.Labels[schema.RELATIONSHIP], 0, "More than 0 relationship labels fetched for RELATIONSHIP type")
						if !assert.Len(t, s.Labels[schema.NODE], 1, "More or less than 1 node labels fetched for NODE type") {
							t.FailNow()
						}

						assert.Equal(t, s.Labels[schema.NODE][0], label, "Fetched node label differs from the one created")
					})

					t.Run("Relationship labels fetched correctly", func(t *testing.T) {
						if !assert.NoError(t, driver.Reset(dbOptions), "Failed to reset DB") {
							return
						}

						label := "RELATIONSHIP_LABEL"

						// Check that labels get read out of the DB
						res := driver.RunQuery(dbOptions, fmt.Sprintf("CREATE ()-[:%s]->()", label))
						assert.Equal(t, driver.GetQueryResultType(res, conf.ErrorMessageRegex), dbms.Valid, "Simple query causes non-valid result type")

						s, err := driver.GetSchema(dbOptions)
						if !assert.NoError(t, err, "Failed to get schema") {
							return
						}

						assert.Len(t, s.Labels[schema.ANY], 1, "More or less than one label fetched")
						assert.Len(t, s.Labels[schema.NODE], 0, "More than 0 node labels fetched")
						if !assert.Len(t, s.Labels[schema.RELATIONSHIP], 1, "More or less than 1 relationship labels fetched") {
							t.FailNow()
						}

						assert.Equal(t, s.Labels[schema.RELATIONSHIP][0], label, "Fetched relationship label differs from the one created")
					})

					// TODO: Table test labels and add tests for properties

				})

				t.Run("Gibberish does not return valid", func(t *testing.T) {
					res := driver.RunQuery(dbOptions, "GIBBERISH")
					assert.NotEqual(t, dbms.Valid, driver.GetQueryResultType(res, conf.ErrorMessageRegex), "Gibberish resulted in a query type of VALID")
				})
			}
		})
	}
}
