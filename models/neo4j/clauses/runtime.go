package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

//go:generate stringer -type=RuntimeType
type RuntimeType int

const (
	noRuntime RuntimeType = iota
	legacy
	pipelined
	slotted
	parallel
	interpreted
)

type Runtime struct {
	runtime RuntimeType
}

// Generate subclauses for OpenCypherRootClause
func (c *Runtime) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.runtime = RuntimeType(seed.GetRandomIntn(int(interpreted) + 1))
	if c.runtime == parallel {
		s.DisallowWriteClauses = true
	}
	return nil
}

func (c Runtime) TemplateString() string {
	if c.runtime == noRuntime {
		return ""
	}
	return "CYPHER runtime = " + c.runtime.String() + " "
}

func (c Runtime) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	return &Runtime{runtime: noRuntime}
}
