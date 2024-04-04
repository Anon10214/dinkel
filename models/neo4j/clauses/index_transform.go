package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Transform an Index to an equivalent by having it evaluate in a (semantically) empty query.
// Creating an index only has an impact on query performance, not its result.
// Thus, omitting the index creation should have no effect on the query's outcome.
func (c *Index) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	// Pretty much an empty query, but sending an empty string causes an exception.
	// Creating an index should have no effect on the graph.
	return helperclauses.CreateStringer("DELETE NULL")
}
