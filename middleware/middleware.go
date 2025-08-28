package middleware

import (
	"github.com/Anon10214/dinkel/dbms"
	"github.com/Anon10214/dinkel/scheduler"
	"github.com/Anon10214/dinkel/scheduler/strategy"
)

// Hooks holds all possible hooks which are called
// by the middleware while fuzzing.
type Hooks struct {
	StrategyHooks strategy.StrategyMiddleware
	DBHooks       dbms.DBMiddleware
}

type Middleware interface {
	// Get the struct of hooks to register
	Hooks() Hooks
}

// RegisterMiddleware takes in a middleware and registers it by modifying the
// passed config.
func RegisterMiddleware(m Middleware, conf *scheduler.Config) {
	hooks := m.Hooks()
	conf.Strategy = strategy.WrapStrategy(conf.Strategy, hooks.StrategyHooks)
	conf.DB = dbms.WrapDB(conf.DB, hooks.DBHooks)
}
