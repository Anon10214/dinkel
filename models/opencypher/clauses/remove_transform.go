package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Transform a RemoveSubclause to an equivalent by repeating one of its targets.
// Removing an element a second time should have no influence on the query result,
// since the target is already removed.
func (c *RemoveSubclause) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	// Removing the same element twice should have the same effect as removing it once
	templateString := "%s, %[1]s"
	if c.isBasecase {
		return helperclauses.CreateAssembler(templateString, subclauses...)
	}
	return helperclauses.CreateAssembler(templateString+", %[2]s", subclauses...)
}
