package neo4j

import (
	"testing"
	"time"

	"github.com/Anon10214/dinkel/dbms"
	"github.com/stretchr/testify/assert"
)

func TestInit_Port(t *testing.T) {
	opts := dbms.DBOptions{
		Host:    "host",
		Port:    nil,
		Timeout: time.Second,
	}

	driver := Driver{}

	t.Run("Default Port", func(t *testing.T) {
		driver.Init(opts)

		assert.Equal(t, "host:7687", driver.driver.Target().Host, "Default port wrong")
	})

	t.Run("User defined Port", func(t *testing.T) {
		port := 123
		opts.Port = &port

		driver.Init(opts)

		assert.Equal(t, "host:123", driver.driver.Target().Host, "Default port wrong")
	})
}
