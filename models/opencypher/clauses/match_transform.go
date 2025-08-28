package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Transform an OptionalWhereClause to an equivalent by having its expression evaluate to true
// if it wasn't generated.
// Since the where clause wasn't generated in the original query, it didn't filter out anything.
// A WHERE true clause also doesn't filter out anything, thus it shouldn't change the outcome of the query.
func (c OptionalWhereClause) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	if !c.willGenerate {
		Conf := schema.ExpressionConfig{
			TargetType:   schema.PropertyValue,
			PropertyType: schema.Boolean,
		}
		return helperclauses.CreateAssembler(
			"WHERE %s",
			&Tautum{conf: Conf},
		)
	}
	return nil
}
