package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Transform a WithClause to an equivalent by including an additional with element.
// Since this element wasn't present the first time the clause got generated,
// it will not be referenced again later, thus it shouldn't change the outcome of the query.
func (c *WithClause) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	return helperclauses.CreateAssembler(
		"WITH %s %s, %s",
		append(subclauses, &WithElement{Name: generateUniqueName(seed, s), Conf: generateExpressionConf(seed)})...,
	)
}
