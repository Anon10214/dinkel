package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/config"
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
					"(CASE (%s) IS NULL WHEN %s THEN null WHEN %s then (%s) %s %s END)",
					subclauses[0], &Tautum{conf: c.Conf}, &Falsum{conf: c.Conf}, subclauses[0].(*helperclauses.ClauseCapturer).Copy(), &CaseExpressionWhen{Conf: c.Conf}, optionalClause(seed, &CaseExpressionElse{Conf: c.Conf}),
				)
			}
			return helperclauses.CreateAssembler(
				"(CASE (%s) IS NOT NULL WHEN %s THEN null WHEN %s then (%s) %s %s END)",
				subclauses[0], &Falsum{conf: c.Conf}, &Tautum{conf: c.Conf}, subclauses[0].(*helperclauses.ClauseCapturer).Copy(), &CaseExpressionWhen{Conf: c.Conf}, optionalClause(seed, &CaseExpressionElse{Conf: c.Conf}),
			)
		}
		// Generic IS (NOT) NULL case expression
		return helperclauses.CreateAssembler(
			"(CASE WHEN (%s) IS NULL THEN null WHEN (%s) IS NOT NULL then (%s) %s %s END)",
			subclauses[0], subclauses[0].(*helperclauses.ClauseCapturer).Copy(), subclauses[0].(*helperclauses.ClauseCapturer).Copy(), &CaseExpressionWhen{IsGeneric: true, Conf: c.Conf}, optionalClause(seed, &CaseExpressionElse{Conf: c.Conf}),
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
								"(-(-(%s)))",
								subclauses...,
							)
						}
					}
					return helperclauses.CreateAssembler(
						"((%s)^1)",
						subclauses...,
					)
				}
				if c.Conf.PropertyType != schema.Float && seed.RandomBoolean() {
					return helperclauses.CreateAssembler(
						seed.RandomStringFromChoice(
							"((%s) + 0)", "(0 + (%s))",
							"((%s) - 0)",
						),
						subclauses...,
					)
				}

				// Don't transform to `x/1` if division is inaccurate
				choices := []string{"((%s) * 1)", "(1 * (%s))"}
				if !config.GetConfig().InaccurateDivision {
					choices = append(choices, "((%s) / 1)")
				}
				return helperclauses.CreateAssembler(
					seed.RandomStringFromChoice(choices...),
					subclauses...,
				)
			}
			if c.Conf.PropertyType == schema.String {
				return helperclauses.CreateAssembler(
					seed.RandomStringFromChoice(
						"((%s)+\"\")", "(\"\"+(%s))",
					),
					subclauses...,
				)
			}
		}
	}
	if c.Conf.IsList {
		return helperclauses.CreateAssembler(
			seed.RandomStringFromChoice(
				"((%s)+[])", "([]+(%s))",
			),
			subclauses...,
		)
	}
	return nil
}
