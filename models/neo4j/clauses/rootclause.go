package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

type RootClause struct{}

// Generate subclauses for OpenCypherRootClause
func (c *RootClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&Runtime{}, &clauses.ReadClause{}}
}
