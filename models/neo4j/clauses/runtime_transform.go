package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

func (c *Runtime) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	newRuntime := RuntimeType(seed.GetRandomIntn(int(interpreted) + 1))
	// Since parallel runtime cannot be run on updating queries
	if newRuntime == parallel {
		newRuntime = noRuntime
	}
	return helperclauses.CreateStringer((&Runtime{
		runtime: newRuntime,
	}).TemplateString())
}
