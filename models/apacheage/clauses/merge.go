package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

// Merge drop-in for apache age
type Merge struct {
	oldAllowOnlyNonNullPropertyExpressions bool
}

// Generate the merge subclauses.
// Different from the OpenCypher implementation, this drop-in doesn't have a
// CreateElementChain as its subclause, instead just a single CreateElement.
func (c *Merge) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.oldAllowOnlyNonNullPropertyExpressions = s.IsInMergeClause
	s.IsInMergeClause = true

	return []translator.Clause{&clauses.CreateElement{}}
}

// TemplateString for the merge drop-in
func (c Merge) TemplateString() string {
	return "MERGE %s"
}

// ModifySchema after drop-in generation
func (c Merge) ModifySchema(s *schema.Schema) {
	s.IsInMergeClause = c.oldAllowOnlyNonNullPropertyExpressions
}
