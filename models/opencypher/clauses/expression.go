package clauses

import (
	"fmt"

	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
	"github.com/sirupsen/logrus"
)

type Expression struct {
	Conf schema.ExpressionConfig
}

// Generate subclauses for Expression
func (c *Expression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	// Decide on types early with some probability (allows transformation earlier in the AST)
	if seed.RandomBoolean() {
		if c.Conf.TargetType == schema.AnyExpression && seed.RandomBoolean() {
			c.Conf.TargetType = schema.ExpressionType(seed.GetRandomIntn(2) + 1)
		}
		if c.Conf.PropertyType == schema.AnyType && seed.RandomBoolean() {
			c.Conf.PropertyType = generatePropertyType(seed)
		}
		if c.Conf.StructuralType == schema.ANY && seed.RandomBoolean() {
			c.Conf.StructuralType = generateStructuralType(seed)
		}
	}

	if c.Conf.TargetType != schema.StructuralValue && !c.Conf.IsList && seed.BooleanWithProbability(0.75) {
		c.Conf.TargetType = schema.PropertyValue
		return []translator.Clause{&PropertyLiteral{Conf: c.Conf}}
	}

	if c.Conf.TargetType != schema.StructuralValue && !c.Conf.MustBeNonNull && !c.Conf.IsList && !c.Conf.IsConstantExpression && c.Conf.PropertyType == schema.Boolean && seed.BooleanWithProbability(0.1) {
		return []translator.Clause{&Predicate{Conf: c.Conf}}
	}

	if c.Conf.IsList {
		return []translator.Clause{&ListExpression{Conf: c.Conf}}
	}

	if seed.BooleanWithProbability(0.25) {
		return []translator.Clause{&FunctionApplicationExpression{Conf: c.Conf}}
	}

	if seed.BooleanWithProbability(0.33) {
		return []translator.Clause{&OperatorApplicationExpression{Conf: c.Conf}}
	}

	if !c.Conf.IsConstantExpression && seed.BooleanWithProbability(0.9) {
		// Subquery expressions
		if c.Conf.TargetType != schema.StructuralValue && !s.IsInMergeClause && !c.Conf.IsList && seed.BooleanWithProbability(0.1) {
			if c.Conf.PropertyType == schema.AnyType {
				if seed.RandomBoolean() {
					c.Conf.PropertyType = schema.Integer
				} else {
					c.Conf.PropertyType = schema.Boolean
				}
			}
			switch c.Conf.PropertyType {
			case schema.Integer:
				return []translator.Clause{&Count{}}
			case schema.Boolean:
				return []translator.Clause{&Exists{}}
			}
		}
		return []translator.Clause{&VariableExpression{Conf: c.Conf}}
	} else if c.Conf.IsConstantExpression {
		return []translator.Clause{&PropertyLiteral{Conf: c.Conf}}
	}

	return []translator.Clause{&CaseExpression{Conf: c.Conf}}
}

func (c Expression) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	seed := seed.GetPregeneratedByteString(make([]byte, 128))
	if c.Conf.PropertyType == schema.AnyType {
		c.Conf.PropertyType = generatePropertyType(seed)
	}
	val := generateLiteral(seed, c.Conf.PropertyType, c.Conf.MustBeNonNull)
	return helperclauses.CreateStringer(fmt.Sprintf("(%s)", val))
}

type VariableExpression struct {
	Conf                       schema.ExpressionConfig
	name                       string
	IsStructuralPropertyAccess bool
}

// Generate subclauses for VariableExpression
func (c *VariableExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	var varMask int
	if c.Conf.IsList {
		varMask |= schema.ListMask
	}
	if c.Conf.MustBeNonNull {
		varMask |= schema.NullableMask
	}

	if c.Conf.TargetType == schema.StructuralValue || (c.Conf.TargetType == schema.AnyExpression && seed.RandomBoolean()) {
		if len(s.StructuralVariablesByType[c.Conf.StructuralType|schema.StructuralType(varMask)]) != 0 {
			availableVars := s.StructuralVariablesByType[c.Conf.StructuralType|schema.StructuralType(varMask)]
			c.name = availableVars[seed.GetRandomIntn(len(availableVars))].Name
			if config.GetConfig().DisallowDeletedWriteTargets && s.DeletedVars[c.name] {
				// Make sure we don't reference a deleted target if the config disallows
				c.name = ""
				return []translator.Clause{&Expression{Conf: c.Conf}}
			}

			// Add the variable to deleted vars if it will be deleted
			if c.Conf.GetsDeleted {
				s.DeletedVars[c.name] = true
				logrus.Tracef("Adding %s to deleted variables", c.name)
			}
			return nil
		} else if c.Conf.TargetType == schema.StructuralValue {
			c.name = "null"
			return nil
		}
	}

	if !c.Conf.MustBeNonNull && seed.RandomBoolean() { // Generate a node or relationship property
		c.IsStructuralPropertyAccess = true

		// Fallback is just a random property name
		c.name = generateName(seed)

		availableProperties := s.Properties[c.Conf.PropertyType|schema.PropertyType(varMask)]
		if len(availableProperties) > 0 {
			targetProperty := availableProperties[seed.GetRandomIntn(len(availableProperties))]
			// If all instances with this property name share the target type
			if c.Conf.PropertyType|schema.PropertyType(varMask) == schema.AnyType || (s.PropertyTypeByName[targetProperty.Name] == c.Conf.PropertyType|schema.PropertyType(varMask)) {
				c.name = targetProperty.Name
			}
		}

		targetType := schema.NODE
		if seed.RandomBoolean() {
			targetType = schema.RELATIONSHIP
		}

		if config.GetConfig().OnlyVariablesAsWriteTarget && len(s.StructuralVariablesByType[targetType]) == 0 {
			c.IsStructuralPropertyAccess = false
			c.name = "null"
			return nil
		}

		return []translator.Clause{&Expression{Conf: schema.ExpressionConfig{
			TargetType:     schema.StructuralValue,
			StructuralType: targetType,
		}}}
	} else if len(s.PropertyVariablesByType[c.Conf.PropertyType|schema.PropertyType(varMask)]) != 0 {
		// Generate a property variable
		availableVars := s.PropertyVariablesByType[c.Conf.PropertyType|schema.PropertyType(varMask)]
		c.name = availableVars[seed.GetRandomIntn(len(availableVars))].Name
		return nil
	}
	return []translator.Clause{&Expression{Conf: c.Conf}}
}

// TemplateString for VariableExpression
func (c VariableExpression) TemplateString() string {
	if c.name == "" {
		return "%s"
	}
	if c.IsStructuralPropertyAccess {
		return "%s." + c.name
	}
	return c.name
}

// WriteTarget represents the target of a DELETE, REMOVE or SET clause.
//
// If NoWriteTargetIndirection is set in the generation config, this clause
// expects that it was already checked that there is an available candidate.
type WriteTarget struct {
	TargetType schema.StructuralType
	// If true, then the write target this will generate will end up being deleted
	GetsDeleted bool
}

// Generate subclauses for WriteTarget
func (c *WriteTarget) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if config.GetConfig().OnlyVariablesAsWriteTarget {
		availableVars := s.StructuralVariablesByType[c.TargetType]
		if len(availableVars) == 0 {
			logrus.Panicf("Attempted to generate a write target with write target indirection disabled but no viable candidate exists for type %d", c.TargetType)
		}
		chosenVar := availableVars[seed.GetRandomIntn(len(availableVars))]
		if c.GetsDeleted {
			s.DeletedVars[chosenVar.Name] = true
		}
		return []translator.Clause{helperclauses.CreateStringer(chosenVar.Name)}
	}
	return []translator.Clause{&Expression{Conf: schema.ExpressionConfig{
		TargetType:     schema.StructuralValue,
		StructuralType: c.TargetType,
		GetsDeleted:    c.GetsDeleted,
	}}}
}
