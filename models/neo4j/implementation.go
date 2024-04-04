package neo4j

import (
	"reflect"

	neo4jclauses "github.com/Anon10214/dinkel/models/neo4j/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

// Implementation for Neo4j
type Implementation struct{}

// GetDropIns returns the clause drop-ins for the Neo4j implementation
func (Implementation) GetDropIns() translator.DropIns {
	return map[reflect.Type]translator.DropIn{
		// Add the Neo4j specific indexes
		reflect.TypeOf(&clauses.Index{}): func(c translator.Clause, seed *seed.Seed, s2 *schema.Schema) translator.Clause {
			return &neo4jclauses.Index{}
		},
	}
}

// GetOpenCypherConfig returns the generation config for the Neo4j implementation
func (Implementation) GetOpenCypherConfig() config.Config {
	return config.Config{
		AdditionalPropertyFunctions:   neo4jPropertyFunctions,
		AdditionalStructuralFunctions: neo4jStructuralFunctions,
		AdditionalMapFunctions:        neo4jMapFunctions,
	}
}

var neo4jPropertyFunctions map[schema.PropertyType][]schema.Function = map[schema.PropertyType][]schema.Function{
	schema.Integer: {
		{
			Name:            "linenumber",
			CanAlwaysBeNull: true,
			InputTypes:      []schema.ExpressionConfig{},
		},
		{
			Name: "toInteger",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Boolean,
				},
			},
		},
		{
			Name:            "toIntegerOrNull",
			CanAlwaysBeNull: true,
			InputTypes: []schema.ExpressionConfig{
				{},
			},
		},
	},
	schema.Float: {
		{
			Name: "haversin",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "point.distance",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Point,
				},
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Point,
				},
			},
		},
		{
			Name: "round",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.PositiveInt32,
				},
			},
		},
		{
			Name:            "toFloatOrNull",
			CanAlwaysBeNull: true,
			InputTypes: []schema.ExpressionConfig{
				{},
			},
		},
	},
	schema.Boolean: {
		{
			Name: "isEmpty",
			InputTypes: []schema.ExpressionConfig{
				{
					IsList: true,
				},
			},
		},
		{
			Name: "isEmpty",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
			},
		},
		{
			Name: "isNaN",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Integer,
				},
			},
		},
		{
			Name: "isNaN",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "point.withinBBox",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Point,
				},
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Point,
				},
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Point,
				},
			},
		},
		{
			Name: "toBoolean",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Integer,
				},
			},
		},
		{
			Name:            "toBooleanOrNull",
			CanAlwaysBeNull: true,
			InputTypes: []schema.ExpressionConfig{
				{},
			},
		},
	},
	schema.String: {
		{
			Name:            "file",
			CanAlwaysBeNull: true,
			InputTypes:      []schema.ExpressionConfig{},
		},
		{
			Name:            "toStringOrNull",
			CanAlwaysBeNull: true,
			InputTypes: []schema.ExpressionConfig{
				{},
			},
		},
	},
	schema.String | schema.PropertyType(schema.ListMask): {
		{
			Name: "split",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
					IsList:       true,
				},
			},
		},
		{
			Name: "toStringList",
			InputTypes: []schema.ExpressionConfig{
				{
					IsList: true,
				},
			},
		},
	},
	schema.Integer | schema.PropertyType(schema.ListMask): {
		{
			Name: "toIntegerList",
			InputTypes: []schema.ExpressionConfig{
				{
					IsList: true,
				},
			},
		},
	},
	schema.Boolean | schema.PropertyType(schema.ListMask): {
		{
			Name: "toBooleanList",
			InputTypes: []schema.ExpressionConfig{
				{
					IsList: true,
				},
			},
		},
	},
	schema.Float | schema.PropertyType(schema.ListMask): {
		{
			Name: "toFloatList",
			InputTypes: []schema.ExpressionConfig{
				{
					IsList: true,
				},
			},
		},
	},
}

var neo4jStructuralFunctions map[schema.StructuralType][]schema.Function = nil

var neo4jMapFunctions []schema.Function = nil
