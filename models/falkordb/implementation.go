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
		// REMOVE often causes a reported crash, ignore for now until fixed
		reflect.TypeOf(&clauses.Remove{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.WriteClause{}
		},

		// Circumvent regression in path variables
		reflect.TypeOf(&clauses.CreateElement{}): func(c translator.Clause, s1 *seed.Seed, s2 *schema.Schema) translator.Clause {
			return &clauses.CreatePathElement{}
		},

		// Subquery expressions are not implemented yet
		reflect.TypeOf(&clauses.Exists{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.Expression{Conf: schema.ExpressionConfig{
				TargetType:   schema.PropertyValue,
				PropertyType: schema.Boolean,
			}}
		},
		reflect.TypeOf(&clauses.Count{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.Expression{Conf: schema.ExpressionConfig{
				TargetType:   schema.PropertyValue,
				PropertyType: schema.Integer,
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

			return helperclauses.CreateAssembler(subexpressions, templateString)
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

		// DELETE null is invalid in FalkorDB
		OnlyVariablesAsWriteTarget: true,

		DisallowedPropertyTypes: []schema.PropertyType{
			// FalkorDB (currently) does not support temporal types.
			schema.Date, schema.Datetime, schema.Duration, schema.LocalDateTime, schema.LocalTime, schema.Time,
			// TODO: FalkorDB does not (yet) support points with x and y, but rather latitude and longitude, adjust generation instead of just disallowing
			schema.Point,
		},
	}
}
