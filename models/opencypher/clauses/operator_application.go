package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

type OperatorApplicationExpression struct {
	Conf           schema.ExpressionConfig
	templateString string
}

// Generate subclauses for OperatorApplicationExpression
func (c *OperatorApplicationExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	// Operators cannot return structural values, generate a different expression
	if c.Conf.TargetType == schema.StructuralValue {
		return []translator.Clause{&Expression{Conf: c.Conf}}
	}

	// Ensure target type is property, not ANY
	c.Conf.TargetType = schema.PropertyValue

	// List concatenation
	if c.Conf.IsList {
		c.templateString = "%s+%s"
		return []translator.Clause{&Expression{Conf: c.Conf}, &Expression{Conf: c.Conf}}
	}

	if c.Conf.PropertyType == schema.AnyType {
		c.Conf.PropertyType = generatePropertyType(seed)
	}
	switch c.Conf.PropertyType {
	case schema.String:
		// String concatenation
		c.templateString = "%s+%s"
		return []translator.Clause{&Expression{Conf: c.Conf}, &Expression{Conf: c.Conf}}
	case schema.Boolean:
		switch seed.GetRandomIntn(3) {
		// Boolean operators
		case 0:
			if seed.BooleanWithProbability(0.25) {
				c.templateString = "NOT %s"
				return []translator.Clause{&Expression{Conf: c.Conf}}
			}
			c.templateString = seed.RandomStringFromChoice("%s AND %s", "%s OR %s", "%s XOR %s")
			return []translator.Clause{&Expression{Conf: c.Conf}, &Expression{Conf: c.Conf}}
		// Comparison operators
		case 1:
			// NULL comparisons
			if seed.RandomBoolean() {
				c.templateString = seed.RandomStringFromChoice("%s IS NULL", "%s IS NOT NULL")
				Conf := generateExpressionConf(seed)
				Conf.IsConstantExpression = c.Conf.IsConstantExpression
				Conf.CanContainAggregatingFunctions = c.Conf.CanContainAggregatingFunctions
				return []translator.Clause{&Expression{Conf: Conf}}
			}
			// General comparisons
			c.templateString = "%s" + seed.RandomStringFromChoice("=", "<>", "<", ">", "<=", ">=") + "%s"
			// If expression cannot be null, subexpressions must be non null and be of the same (property) type, disallow structural
			if c.Conf.MustBeNonNull {
				subexpressionConf := generateExpressionConf(seed)
				// Comparing points or durations or durations often results in NULL
				if subexpressionConf.PropertyType == schema.Point || subexpressionConf.PropertyType == schema.Duration {
					subexpressionConf.PropertyType = schema.Integer
				}
				subexpressionConf.TargetType = schema.PropertyValue
				subexpressionConf.MustBeNonNull = true
				subexpressionConf.IsConstantExpression = c.Conf.IsConstantExpression
				subexpressionConf.CanContainAggregatingFunctions = c.Conf.CanContainAggregatingFunctions
				subexpressionConf.IsList = false // Comparing lists leads to NULL
				return []translator.Clause{&Expression{Conf: subexpressionConf}, &Expression{Conf: subexpressionConf}}
			}
			firstConf := generateExpressionConf(seed)
			firstConf.IsConstantExpression = c.Conf.IsConstantExpression
			firstConf.CanContainAggregatingFunctions = c.Conf.CanContainAggregatingFunctions
			secondConf := generateExpressionConf(seed)
			secondConf.IsConstantExpression = c.Conf.IsConstantExpression
			secondConf.CanContainAggregatingFunctions = c.Conf.CanContainAggregatingFunctions
			return []translator.Clause{&Expression{Conf: firstConf}, &Expression{Conf: secondConf}}
		// String-specific comparison operators
		case 2:
			c.Conf.PropertyType = schema.String
			c.templateString = seed.RandomStringFromChoice("%s STARTS WITH %s", "%s ENDS WITH %s", "%s CONTAINS %s", "%s =~ %s")
			return []translator.Clause{&Expression{Conf: c.Conf}, &Expression{Conf: c.Conf}}
		}
	case schema.Float:
		// Generate a power expression, as they always evaluate to floats
		if seed.BooleanWithProbability(0.25) {
			c.templateString = "%s^%s"
			return []translator.Clause{&Expression{Conf: c.Conf}, &Expression{Conf: c.Conf}}
		}
		fallthrough
	case schema.Integer:
		// Have to escape modulo operator
		c.templateString = "%s" + seed.RandomStringFromChoice("+", "-", "*", "/", "%%") + "%s"
		return []translator.Clause{&Expression{Conf: c.Conf}, &Expression{Conf: c.Conf}}
		// TODO: Add support for temporal type applications
	}
	return []translator.Clause{&Expression{Conf: c.Conf}}
}

// TemplateString for OperatorApplicationExpression
func (c OperatorApplicationExpression) TemplateString() string {
	if c.templateString == "" {
		return "%s"
	}
	return "(" + c.templateString + ")"
}
