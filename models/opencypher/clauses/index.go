package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

// The Index type serves only to be intercepted by drop-ins of implementations.
// Every model should have its own index implementation.
//
// If it is not intercepted, it simply returns a root clause, generating a normal query.
type Index struct{}

// Generate subclauses for Index
func (c *Index) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&ReadClause{}}
}
