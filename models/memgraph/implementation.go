package memgraph

import (
	"reflect"

	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

// Implementation for memgraph
type Implementation struct{}

// GetDropIns returns the clause drop-ins for the memgraph implementation
func (Implementation) GetDropIns() translator.DropIns {
	return map[reflect.Type]translator.DropIn{
		// https://github.com/memgraph/memgraph/issues/887
		reflect.TypeOf(&clauses.Remove{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.WriteClause{}
		},
		reflect.TypeOf(&clauses.Delete{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.WriteClause{}
		},
		reflect.TypeOf(&clauses.Set{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.WriteClause{}
		},

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
			}}
		},
		reflect.TypeOf(&clauses.Count{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.Expression{Conf: schema.ExpressionConfig{
				TargetType:   schema.PropertyValue,
				PropertyType: schema.Integer,
			}}
		},
	}
}

// GetOpenCypherConfig returns the generation config for the memgraph implementation
func (Implementation) GetOpenCypherConfig() config.Config {
	return config.Config{
		DisallowedPropertyTypes: []schema.PropertyType{
			// memgraph (currently) does not support temporal types.
			schema.Date, schema.Datetime, schema.Duration, schema.LocalDateTime, schema.LocalTime, schema.Time,
			// memgraph (currently) does not support points.
			schema.Point,
		},
	}
}
