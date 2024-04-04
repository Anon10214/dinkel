package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Transform an OperatorApplicationExpression to an equivalent expression.
func (c OperatorApplicationExpression) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	switch c.templateString {
	// De morgan
	case "%s AND %s":
		return helperclauses.CreateAssembler(subclauses, "(NOT (NOT (%s) OR NOT (%s)))")
	case "%s OR %s":
		return helperclauses.CreateAssembler(subclauses, "(NOT (NOT (%s) AND NOT (%s)))")

	case "%s IS NULL":
		return helperclauses.CreateAssembler(subclauses, "(NOT (%s IS NOT NULL))")
	case "%s IS NOT NULL":
		return helperclauses.CreateAssembler(subclauses, "(NOT (%s IS NULL))")
	}
	return nil
}
