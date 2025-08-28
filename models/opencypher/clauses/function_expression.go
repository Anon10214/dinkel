package clauses

import (
	"fmt"
	"strings"

	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

// A FunctionApplicationExpression generates a function application.
//
// It chooses one of OpenCypher's required functions or one of the
// additional functions from the generation config.
// It never generates a function in the list of forbidden functions
// defined in the generation config.
type FunctionApplicationExpression struct {
	Conf   schema.ExpressionConfig
	target *schema.Function
}

// Generate subclauses for FunctionApplicationExpression
func (c *FunctionApplicationExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if c.Conf.IsConstantExpression {
		return []translator.Clause{&Expression{Conf: c.Conf}}
	}
	var mask int

	if c.Conf.IsList {
		mask = schema.ListMask
	}

	// Set the concrete target type with some probability if undecided, otherwise use functions that return MAPs if allowed
	if c.Conf.TargetType == schema.AnyExpression && (!c.Conf.AllowMaps || seed.BooleanWithProbability(0.75)) {
		c.Conf.TargetType = schema.ExpressionType(seed.GetRandomIntn(2) + 1)
	}

	genConf := config.GetConfig()

	var targets []schema.Function
	var found bool
	if c.Conf.TargetType == schema.PropertyValue {
		propertyType := c.Conf.PropertyType | schema.PropertyType(mask)

		targets1, found1 := propertyFunctionsByReturnType[propertyType]
		targets2, found2 := genConf.AdditionalPropertyFunctions[propertyType]

		targets = append(targets1, targets2...)
		found = found1 || found2
	} else if c.Conf.TargetType == schema.StructuralValue {
		structuralType := c.Conf.StructuralType | schema.StructuralType(mask)

		targets1, found1 := structuralFunctionsByReturnType[structuralType]
		targets2, found2 := genConf.AdditionalStructuralFunctions[structuralType]

		targets = append(targets1, targets2...)
		found = found1 || found2
	} else {
		found = true
		targets = append(mapFunctions, genConf.AdditionalMapFunctions...)
	}

	// Use aggregation functions if this is a return expression with some probability
	if c.Conf.TargetType != schema.StructuralValue && c.Conf.CanContainAggregatingFunctions && !s.DisallowAggregateFunctions && seed.BooleanWithProbability(0.5) {
		// Return COUNT with some probability
		if c.Conf.PropertyType == schema.Integer && !c.Conf.IsList && seed.RandomBoolean() {
			c.target = nil
			return []translator.Clause{&CountFunction{}}
		}
		propertyType := c.Conf.PropertyType | schema.PropertyType(mask)

		aggTargets1, aggFound1 := aggFunctions[propertyType]
		aggTargets2, aggFound2 := genConf.AdditionalAggregationFunctions[propertyType]

		aggTargets := append(aggTargets1, aggTargets2...)
		aggFound := aggFound1 || aggFound2

		// Update target if found
		if aggFound {
			targets = aggTargets
			found = aggFound
			// Make sure subexpressions don't contain aggregation functions
			c.Conf.CanContainAggregatingFunctions = false
		}
	}

	if !found {
		return []translator.Clause{&Expression{Conf: c.Conf}}
	}
	targetFunction := targets[seed.GetRandomIntn(len(targets))]

	// If function can always return null but conf disallows null
	if targetFunction.CanAlwaysBeNull && c.Conf.MustBeNonNull {
		return []translator.Clause{&Expression{Conf: c.Conf}}
	}

	// Make sure no disallowed functions get generated
	for _, disallowedName := range genConf.DisallowedFunctions {
		if disallowedName == targetFunction.Name {
			return []translator.Clause{&Expression{Conf: c.Conf}}
		}
	}

	c.target = &targetFunction

	subclauses := []translator.Clause{}
	for _, expr := range targetFunction.InputTypes {
		expr.CanContainAggregatingFunctions = c.Conf.CanContainAggregatingFunctions
		// Don't generate function expression if a target type is structural and expression must be non null, as structural values can always evaluate to null
		if expr.TargetType == schema.StructuralValue && c.Conf.MustBeNonNull {
			c.target = nil
			return []translator.Clause{&Expression{Conf: c.Conf}}
		}
		if c.Conf.MustBeNonNull {
			// Make sure expr.TargetType isn't ANY
			expr.TargetType = schema.PropertyValue
			expr.MustBeNonNull = c.Conf.MustBeNonNull
		}
		subclauses = append(subclauses, &Expression{Conf: expr})
	}
	return subclauses
}

// TemplateString for FunctionApplicationExpression
func (c FunctionApplicationExpression) TemplateString() string {
	if c.target == nil {
		return "%s"
	}
	args := ""
	if len(c.target.InputTypes) != 0 {
		args = "%s" + strings.Repeat(", %s", len(c.target.InputTypes)-1)
	}
	return c.target.Name + "(" + args + ")"
}

type CountFunction struct {
	distinct bool
	asterisk bool
}

// Generate subclauses for CountFunction
func (c *CountFunction) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.distinct = seed.BooleanWithProbability(0.33)
	if !c.distinct && seed.RandomBoolean() {
		c.asterisk = true
		return nil
	}
	return []translator.Clause{&Expression{}}
}

// TemplateString for CountFunction
func (c CountFunction) TemplateString() string {
	distinct := ""
	if c.distinct {
		distinct = "DISTINCT "
	}
	expression := "%s"
	if c.asterisk {
		expression = "*"
	}

	return fmt.Sprintf("COUNT(%s%s)", distinct, expression)
}

var propertyFunctionsByReturnType = map[schema.PropertyType][]schema.Function{
	schema.Integer: {
		{
			Name: "abs",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Integer,
				},
			},
		},
		{
			Name: "length",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:     schema.StructuralValue,
					StructuralType: schema.PATH,
				},
			},
		},
		{
			Name: "sign",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Integer,
				},
			},
		},
		{
			Name: "sign",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "size",
			InputTypes: []schema.ExpressionConfig{
				{
					IsList: true,
				},
			},
		},
		{
			Name: "size",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
			},
		},
		{
			Name:            "toInteger",
			CanAlwaysBeNull: true,
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
			},
		},
		{
			Name: "toInteger",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Integer,
				},
			},
		},
		{
			Name: "toInteger",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
	},
	schema.Float: {
		{
			Name: "abs",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "acos",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "asin",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "atan",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "atan2",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "ceil",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "cos",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "cot",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "degrees",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name:       "e",
			InputTypes: []schema.ExpressionConfig{},
		},
		{
			Name: "exp",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "floor",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "log",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "log10",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name:       "pi",
			InputTypes: []schema.ExpressionConfig{},
		},
		{
			Name: "radians",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
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
			},
		},
		{
			Name: "sin",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "sqrt",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "tan",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name:            "toFloat",
			CanAlwaysBeNull: true,
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
			},
		},
		{
			Name: "toFloat",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Integer,
				},
			},
		},
		{
			Name: "toFloat",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
	},
	schema.Boolean: {
		{
			Name:            "toBoolean",
			CanAlwaysBeNull: true,
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
			},
		},
		{
			Name: "toBoolean",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Boolean,
				},
			},
		},
	},
	schema.AnyType: {
		{
			Name:            "head",
			CanAlwaysBeNull: true,
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType: schema.PropertyValue,
					IsList:     true,
				},
			},
		},
		{
			Name:            "last",
			CanAlwaysBeNull: true,
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType: schema.PropertyValue,
					IsList:     true,
				},
			},
		},
	},
	schema.String: {
		{
			Name: "left",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
				{
					TargetType:    schema.PropertyValue,
					PropertyType:  schema.PositiveInteger,
					MustBeNonNull: true,
				},
			},
		},
		{
			Name: "ltrim",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
			},
		},
		{
			Name: "replace",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
			},
		},
		{
			Name: "reverse",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
			},
		},
		{
			Name: "right",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
				{
					TargetType:    schema.PropertyValue,
					PropertyType:  schema.PositiveInteger,
					MustBeNonNull: true,
				},
			},
		},
		{
			Name: "rtrim",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
			},
		},
		{
			Name: "substring",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
				{
					TargetType:    schema.PropertyValue,
					PropertyType:  schema.PositiveInt32,
					MustBeNonNull: true,
				},
			},
		},
		{
			Name: "substring",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
				{
					TargetType:    schema.PropertyValue,
					PropertyType:  schema.PositiveInt32,
					MustBeNonNull: true,
				},
				{
					TargetType:    schema.PropertyValue,
					PropertyType:  schema.PositiveInt32,
					MustBeNonNull: true,
				},
			},
		},
		{
			Name: "toLower",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
			},
		},
		{
			Name: "toString",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType: schema.PropertyValue,
				},
			},
		},
		{
			Name: "toUpper",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
			},
		},
		{
			Name: "trim",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.String,
				},
			},
		},
		{
			Name: "type",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:     schema.StructuralValue,
					StructuralType: schema.RELATIONSHIP,
				},
			},
		},
	},
	schema.String | schema.PropertyType(schema.ListMask): {
		{
			Name: "keys",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:     schema.StructuralValue,
					StructuralType: schema.NODE,
				},
			},
		},
		{
			Name: "keys",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:     schema.StructuralValue,
					StructuralType: schema.RELATIONSHIP,
				},
			},
		},
		{
			Name: "labels",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:     schema.StructuralValue,
					StructuralType: schema.NODE,
				},
			},
		},
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
				},
			},
		},
	},
	schema.AnyType | schema.PropertyType(schema.ListMask): {
		{
			Name: "reverse",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType: schema.PropertyValue,
					IsList:     true,
				},
			},
		},
		{
			Name: "tail",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType: schema.PropertyValue,
					IsList:     true,
				},
			},
		},
	},
}

var structuralFunctionsByReturnType = map[schema.StructuralType][]schema.Function{
	schema.NODE: {
		{
			Name: "endNode",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:     schema.StructuralValue,
					StructuralType: schema.RELATIONSHIP,
				},
			},
		},
		{
			Name: "startNode",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:     schema.StructuralValue,
					StructuralType: schema.RELATIONSHIP,
				},
			},
		},
	},
	schema.NODE | schema.StructuralType(schema.ListMask): {
		{
			Name: "nodes",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:     schema.StructuralValue,
					StructuralType: schema.PATH,
				},
			},
		},
	},
	schema.RELATIONSHIP | schema.StructuralType(schema.ListMask): {
		{
			Name: "relationships",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:     schema.StructuralValue,
					StructuralType: schema.PATH,
				},
			},
		},
	},
}

// Functions that return maps
var mapFunctions = []schema.Function{
	{
		Name: "properties",
		InputTypes: []schema.ExpressionConfig{
			{
				TargetType:     schema.StructuralValue,
				StructuralType: schema.NODE,
			},
		},
	},
	{
		Name: "properties",
		InputTypes: []schema.ExpressionConfig{
			{
				TargetType:     schema.StructuralValue,
				StructuralType: schema.RELATIONSHIP,
			},
		},
	},
}

// Aggregation functions. Can only be used in return statements
var aggFunctions = map[schema.PropertyType][]schema.Function{
	schema.AnyType | schema.PropertyType(schema.ListMask): {
		{
			Name: "collect",
			InputTypes: []schema.ExpressionConfig{
				{},
			},
		},
	},
	schema.AnyType: {
		{
			Name: "max",
			InputTypes: []schema.ExpressionConfig{
				{},
			},
		},
		{
			Name: "min",
			InputTypes: []schema.ExpressionConfig{
				{},
			},
		},
	},
	schema.Integer: {
		{
			Name: "sum",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Integer,
				},
			},
		},
		{
			Name: "avg",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Integer,
				},
			},
		},
		{
			Name: "percentileDisc",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Integer,
				},
				{
					TargetType:    schema.PropertyValue,
					PropertyType:  schema.Percentile,
					MustBeNonNull: true,
				},
			},
		},
	},
	schema.Float: {
		{
			Name: "sum",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "avg",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "percentileCont",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
				{
					TargetType:    schema.PropertyValue,
					PropertyType:  schema.Percentile,
					MustBeNonNull: true,
				},
			},
		},
		{
			Name: "percentileDisc",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
				{
					TargetType:    schema.PropertyValue,
					PropertyType:  schema.Percentile,
					MustBeNonNull: true,
				},
			},
		},
		{
			Name: "stdev",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
		{
			Name: "stdevp",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Float,
				},
			},
		},
	},
	schema.Duration: {
		{
			Name: "sum",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Duration,
				},
			},
		},
		{
			Name: "avg",
			InputTypes: []schema.ExpressionConfig{
				{
					TargetType:   schema.PropertyValue,
					PropertyType: schema.Duration,
				},
			},
		},
	},
}
