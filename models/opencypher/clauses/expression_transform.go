package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Transform an Expression to an equivalent through various means, depending on its type.
func (c *Expression) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	if seed.BooleanWithProbability(0.1) {
		// Can cause more rows to be returned even if no bug is present
		c.Conf.CanContainAggregatingFunctions = false
		// Simple IS (NOT) NULL case expression
		if seed.RandomBoolean() {
			c.Conf.TargetType = schema.AnyExpression
			c.Conf.PropertyType = schema.AnyType
			c.Conf.StructuralType = schema.ANY
			if seed.RandomBoolean() {
				return helperclauses.CreateAssembler(
					[]translator.Clause{subclauses[0], subclauses[0], &CaseExpressionWhen{Conf: c.Conf}, optionalClause(seed, &CaseExpressionElse{Conf: c.Conf})},
					"(CASE (%s) IS NULL WHEN true THEN null WHEN false then (%s) %s %s END)",
				)
			}
			return helperclauses.CreateAssembler(
				[]translator.Clause{subclauses[0], subclauses[0], &CaseExpressionWhen{Conf: c.Conf}, optionalClause(seed, &CaseExpressionElse{Conf: c.Conf})},
				"(CASE (%s) IS NOT NULL WHEN false THEN null WHEN true then (%s) %s %s END)",
			)
		}
		// Generic IS (NOT) NULL case expression
		return helperclauses.CreateAssembler(
			[]translator.Clause{subclauses[0], subclauses[0], subclauses[0], &CaseExpressionWhen{IsGeneric: true, Conf: c.Conf}, optionalClause(seed, &CaseExpressionElse{Conf: c.Conf})},
			"(CASE WHEN (%s) IS NULL THEN null WHEN (%s) IS NOT NULL then (%s) %s %s END)",
		)
	}

	if c.Conf.TargetType == schema.PropertyValue {
		if !c.Conf.IsList {
			if c.Conf.PropertyType == schema.Float || c.Conf.PropertyType == schema.Integer {
				if c.Conf.PropertyType == schema.Float && seed.RandomBoolean() {
					if subclause, ok := subclauses[0].(*PropertyLiteral); ok {
						// -(-(-0.0)) != -0.0
						if subclause.value != "-0.0" && seed.BooleanWithProbability(0.5) {
							return helperclauses.CreateAssembler(
								subclauses,
								"(-(-(%s)))",
							)
						}
					}
					return helperclauses.CreateAssembler(
						subclauses,
						"(%s)^1",
					)
				}
				if c.Conf.PropertyType != schema.Float && seed.RandomBoolean() {
					return helperclauses.CreateAssembler(
						subclauses,
						seed.RandomStringFromChoice(
							"((%s) + 0)", "(0 + (%s))",
							"((%s) - 0)",
						),
					)
				}
				return helperclauses.CreateAssembler(
					subclauses,
					seed.RandomStringFromChoice(
						"((%s) * 1)", "(1 * (%s))",
						"((%s) / 1)",
					),
				)
			}
			if c.Conf.PropertyType == schema.String {
				return helperclauses.CreateAssembler(
					subclauses,
					seed.RandomStringFromChoice(
						"((%s)+\"\")", "(\"\"+(%s))",
					),
				)
			}
		}
	}
	if c.Conf.IsList {
		return helperclauses.CreateAssembler(
			subclauses,
			seed.RandomStringFromChoice(
				"((%s)+[])", "([]+(%s))",
			),
		)
	}
	return nil
}
