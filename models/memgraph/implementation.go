package memgraph

import (
	"reflect"

	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Implementation for memgraph
type Implementation struct{}

// GetDropIns returns the clause drop-ins for the memgraph implementation
func (Implementation) GetDropIns() translator.DropIns {
	return map[reflect.Type]translator.DropIn{
		// List comprehension currently unsupported
		reflect.TypeOf(&clauses.ListComprehension{}): func(c translator.Clause, seed *seed.Seed, s *schema.Schema) translator.Clause {
			clause := c.(*clauses.ListComprehension)
			return &clauses.Expression{Conf: clause.Conf}
		},

		// Subquery expressions are not implemented yet
		reflect.TypeOf(&clauses.Exists{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
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

		reflect.TypeOf(&clauses.ExistingProperty{}): func(c translator.Clause, seed *seed.Seed, s *schema.Schema) translator.Clause {
			return &clauses.NewProperty{}
		},

		reflect.TypeOf(&clauses.PropertyLiteral{}): func(c translator.Clause, s1 *seed.Seed, s2 *schema.Schema) translator.Clause {
			clause := c.(*clauses.PropertyLiteral)
			if clause.Conf.PropertyType == schema.Float {
				return helperclauses.CreateStringer("1.0")
			}
			return clause
		},
	}
}

// GetOpenCypherConfig returns the generation config for the memgraph implementation
func (Implementation) GetOpenCypherConfig() config.Config {
	return config.Config{
		OnlyVariablesAsWriteTarget:      true,
		AsteriskNeedsTargets:            true,
		DisallowMatchAfterOptionalMatch: true,
		DisallowedPropertyTypes: []schema.PropertyType{
			// memgraph (currently) does not support temporal types.
			schema.Date, schema.Datetime, schema.Duration, schema.LocalDateTime, schema.LocalTime, schema.Time,
			// memgraph (currently) does not support points.
			schema.Point,
			// Causes too many timeouts and headaches
			schema.Float,
		},
		DisallowedFunctions: []string{"range", "length", "reverse", "cot", "degrees", "radians", "percentileDisc", "percentileCont",
			// Easily paralyzes the server
			"replace"},
	}
}
