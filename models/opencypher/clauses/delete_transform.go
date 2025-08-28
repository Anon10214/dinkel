package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Transform a DeleteClause to an equivalent by changing a DELETE clause
// to be a DETACH DELETE clause instead.
// Since delete can only be called on relationships or nodes without relationships,
// changing it to a DETACH DELETE should have no effect on the query result.
func (c *DeleteClause) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	// DELETE x => DETACH DELETE x
	if !c.useDetach {
		return helperclauses.CreateAssembler(
			"DETACH DELETE %s",
			subclauses...,
		)
	}
	return nil
}
