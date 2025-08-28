package apacheage

import (
	"reflect"

	apacheageclauses "github.com/Anon10214/dinkel/models/apacheage/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Implementation for apache age
type Implementation struct{}

// GetDropIns returns the clause drop-ins for the apache age implementation
func (Implementation) GetDropIns() translator.DropIns {
	return translator.DropIns{
		// Annoying requirement that ORDER BY in WITH has to reference variables in WITH - unnecessary to accomodate
		reflect.TypeOf(&clauses.OptionalOrderBy{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.EmptyClause{}
		},

		// Indirection has to be on a variable
		reflect.TypeOf(&clauses.VariableExpression{}): func(c translator.Clause, seed *seed.Seed, s *schema.Schema) translator.Clause {
			subclauses := c.Generate(seed, s)
			expr := c.(*clauses.VariableExpression)
			if expr.IsStructuralPropertyAccess {
				if len(s.StructuralVariablesByType[schema.NODE])+len(s.StructuralVariablesByType[schema.RELATIONSHIP]) == 0 {
					return helperclauses.CreateStringer("null")
				}
				if len(s.StructuralVariablesByType[schema.RELATIONSHIP]) == 0 || (seed.RandomBoolean() && len(s.StructuralVariablesByType[schema.NODE]) != 0) {
					return helperclauses.CreateAssembler(expr.TemplateString(), &clauses.WriteTarget{TargetType: schema.NODE})
				}
				return helperclauses.CreateAssembler(expr.TemplateString(), &clauses.WriteTarget{TargetType: schema.RELATIONSHIP})
			}
			return helperclauses.CreateAssembler(expr.TemplateString(), subclauses...)
		},

		// List comprehension currently unsupported
		reflect.TypeOf(&clauses.SimpleCaseExpression{}): func(c translator.Clause, seed *seed.Seed, s *schema.Schema) translator.Clause {
			clause := c.(*clauses.SimpleCaseExpression)
			if clause.Conf.PropertyType == schema.AnyType {
				clause.Conf.PropertyType = schema.Integer
			}
			return clause
		},

		// List comprehension currently unsupported
		reflect.TypeOf(&clauses.ListComprehension{}): func(c translator.Clause, seed *seed.Seed, s *schema.Schema) translator.Clause {
			clause := c.(*clauses.ListComprehension)
			return &clauses.Expression{Conf: clause.Conf}
		},

		// Predicates currently unsupported
		reflect.TypeOf(&clauses.Predicate{}): func(c translator.Clause, seed *seed.Seed, s *schema.Schema) translator.Clause {
			return &clauses.Expression{Conf: c.(*clauses.Predicate).Conf}
		},

		reflect.TypeOf(&clauses.MergeClause{}): func(c translator.Clause, seed *seed.Seed, s *schema.Schema) translator.Clause {
			return &apacheageclauses.Merge{}
		},

		// FOREACH is not implemented yet
		reflect.TypeOf(&clauses.Foreach{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.ReadClause{}
		},
		// UNWIND is not implemented yet
		reflect.TypeOf(&clauses.Unwind{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.ReadClause{}
		},

		// CALL subqueries are not implemented yet
		reflect.TypeOf(&clauses.CallSubquery{}): func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
			return &clauses.ReadClause{}
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

		// Apache AGE doesn't allow sharing labels between vertices and edges
		reflect.TypeOf(&clauses.ExistingLabel{}): func(c translator.Clause, seed *seed.Seed, s *schema.Schema) translator.Clause {
			return &apacheageclauses.ExistingLabel{LabelType: c.(*clauses.ExistingLabel).LabelType}
		},

		// Label matches can only use the old label match syntax
		reflect.TypeOf(&clauses.LabelMatch{}): func(c translator.Clause, seed *seed.Seed, s *schema.Schema) translator.Clause {
			s.UseNewLabelMatchType = new(bool)
			return c
		},

		reflect.TypeOf(&clauses.Labels{}): func(c translator.Clause, s1 *seed.Seed, s2 *schema.Schema) translator.Clause {
			return &clauses.Label{LabelType: c.(*clauses.Labels).LabelType}
		},
		reflect.TypeOf(&clauses.LabelMatch{}): func(c translator.Clause, s1 *seed.Seed, s2 *schema.Schema) translator.Clause {
			return &clauses.LabelName{LabelType: c.(*clauses.LabelMatch).LabelType}
		},

		reflect.TypeOf(&clauses.RemoveLabelExpression{}): func(c translator.Clause, s1 *seed.Seed, s2 *schema.Schema) translator.Clause {
			return &clauses.RemovePropertyExpression{}
		},
		reflect.TypeOf(&clauses.SetLabelExpression{}): func(c translator.Clause, s1 *seed.Seed, s2 *schema.Schema) translator.Clause {
			return &clauses.SetPropertyExpression{}
		},

		// Can only return a single column at once with the SQL foundation given
		reflect.TypeOf(&clauses.ReturnElementChain{}): func(c translator.Clause, s1 *seed.Seed, s2 *schema.Schema) translator.Clause {
			return &clauses.ReturnElement{}
		},
	}
}

// GetOpenCypherConfig returns the generation config for the apache age implementation
func (Implementation) GetOpenCypherConfig() config.Config {
	return config.Config{
		OnlyVariablesAsWriteTarget: true,

		AsteriskNeedsTargets: true,

		DisallowMatchAfterOptionalMatch: true,

		DisallowedFunctions: []string{"substring"},

		DisallowedPropertyTypes: []schema.PropertyType{
			// apache age (currently) does not support temporal types.
			schema.Date, schema.Datetime, schema.Duration, schema.LocalDateTime, schema.LocalTime, schema.Time,
			// apache age (currently) does not support points.
			schema.Point,
		},
	}
}
