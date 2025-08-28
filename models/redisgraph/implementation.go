package redisgraph

import (
	"reflect"

	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Implementation for RedisGraph
type Implementation struct{}

// GetDropIns returns the clause drop-ins for the RedisGraph implementation
func (Implementation) GetDropIns() translator.DropIns {
	return map[reflect.Type]translator.DropIn{
		// Don't generate any WITH * clauses
		reflect.TypeOf(&clauses.WithClause{}): func(c translator.Clause, s1 *seed.Seed, s2 *schema.Schema) translator.Clause {
			return helperclauses.CreateAssembler(
				"WITH %s",
				&clauses.WithElementChain{},
			)
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

		// List comprehension currently unsupported
		reflect.TypeOf(&clauses.ListComprehension{}): func(c translator.Clause, seed *seed.Seed, s *schema.Schema) translator.Clause {
			clause := c.(*clauses.ListComprehension)
			return &clauses.Expression{Conf: clause.Conf}
		},

		// CALL subqueries are not implemented yet
		reflect.TypeOf(&clauses.CallSubquery{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.ReadClause{}
		},
	}
}

// GetOpenCypherConfig returns the generation config for the RedisGraph implementation
func (Implementation) GetOpenCypherConfig() config.Config {
	return config.Config{
		DisallowedFunctions: []string{
			// Range causes crashes too often, cluttering bug reports
			"range",
		},

		// DELETE null is invalid in RedisGraph
		OnlyVariablesAsWriteTarget: true,

		InaccurateDivision: true,

		DisallowedPropertyTypes: []schema.PropertyType{
			// RedisGraph (currently) does not support temporal types.
			schema.Date, schema.Datetime, schema.Duration, schema.LocalDateTime, schema.LocalTime, schema.Time,
			// TODO: RedisGraph does not (yet) support points with x and y, but rather latitude and longitude, adjust generation instead of just disallowing
			schema.Point,
		},
	}
}
