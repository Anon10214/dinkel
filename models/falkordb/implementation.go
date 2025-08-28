package falkordb

import (
	"reflect"
	"strings"

	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Implementation for FlakorDB
type Implementation struct{}

// GetDropIns returns the clause drop-ins for the FlakorDB implementation
func (Implementation) GetDropIns() translator.DropIns {
	return map[reflect.Type]translator.DropIn{
		reflect.TypeOf(&clauses.StringLiteral{}): func(c translator.Clause, seed *seed.Seed, s2 *schema.Schema) translator.Clause {
			// Just use standard ASCII with some probability
			if seed.RandomBoolean() {
				return c
			}

			// Use standard ascii with the probability of adding a unicode code point
			res := `"`
			for seed.RandomBoolean() {
				var nextChar string
				if seed.RandomBoolean() {
					nextChar = string(rune(seed.GetRandomIntn(127-32) + 32))
				} else {
					nextChar = string(rune(int32(seed.GetRandomInt64())))
				}
				res += strings.ReplaceAll(strings.ReplaceAll(nextChar, `"`, ""), `\`, "")
			}
			res += `"`
			return helperclauses.CreateStringer(res)
		},

		// Don't generate any WITH * clauses
		reflect.TypeOf(&clauses.WithClause{}): func(c translator.Clause, s1 *seed.Seed, s2 *schema.Schema) translator.Clause {
			return helperclauses.CreateAssembler(
				"WITH %s",
				&clauses.WithElementChain{},
			)
		},
		// REMOVE often causes a reported crash, ignore for now until fixed
		reflect.TypeOf(&clauses.Remove{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.WriteClause{}
		},

		// Circumvent regression in path variables
		reflect.TypeOf(&clauses.CreateElement{}): func(c translator.Clause, s1 *seed.Seed, s2 *schema.Schema) translator.Clause {
			return &clauses.CreatePathElement{}
		},

		// Subquery expressions are not implemented yet
		reflect.TypeOf(&clauses.Exists{}): func(c translator.Clause, s1 *seed.Seed, s2 *schema.Schema) translator.Clause {
			return &clauses.Expression{Conf: schema.ExpressionConfig{
				TargetType:   schema.PropertyValue,
				PropertyType: schema.Boolean,

				// Be conservative
				MustBeNonNull:                  true,
				IsList:                         false,
				IsConstantExpression:           true,
				CanContainAggregatingFunctions: false,
				AllowMaps:                      false,
			}}
		},
		reflect.TypeOf(&clauses.Count{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.Expression{Conf: schema.ExpressionConfig{
				TargetType:   schema.PropertyValue,
				PropertyType: schema.Integer,

				// Be conservative
				MustBeNonNull:                  true,
				IsList:                         false,
				IsConstantExpression:           true,
				CanContainAggregatingFunctions: false,
				AllowMaps:                      false,
			}}
		},

		// Label matches can only use the old label match syntax
		reflect.TypeOf(&clauses.LabelMatch{}): func(c translator.Clause, seed *seed.Seed, s *schema.Schema) translator.Clause {
			s.UseNewLabelMatchType = new(bool)
			return c
		},

		// FalkorDB hasn't yet implemented regex comparisons
		// So we just replace =~ with CONTAINS
		reflect.TypeOf(&clauses.OperatorApplicationExpression{}): func(c translator.Clause, s1 *seed.Seed, s2 *schema.Schema) translator.Clause {
			expression := c.(*clauses.OperatorApplicationExpression)

			subexpressions := expression.Generate(s1, s2)
			templateString := strings.Replace(expression.TemplateString(), "=~", "CONTAINS", -1)

			return helperclauses.CreateAssembler(templateString, subexpressions...)
		},
	}
}

// GetOpenCypherConfig returns the generation config for the FlakorDB implementation
func (Implementation) GetOpenCypherConfig() config.Config {
	return config.Config{
		DisallowedFunctions: []string{
			// Range causes crashes too often, cluttering bug reports
			"range",
		},

		// Maybe fixed when #629 closed?
		InaccurateDivision: true,

		// DELETE null is invalid in FalkorDB
		OnlyVariablesAsWriteTarget: true,

		DisallowMatchAfterOptionalMatch: true,

		DisallowedPropertyTypes: []schema.PropertyType{
			// FalkorDB (currently) does not support temporal types.
			schema.Date, schema.Datetime, schema.Duration, schema.LocalDateTime, schema.LocalTime, schema.Time,
			// Causes too many timeouts and headaches
			schema.Float,
			// TODO: FalkorDB does not (yet) support points with x and y, but rather latitude and longitude, adjust generation instead of just disallowing
			schema.Point,
		},
	}
}
