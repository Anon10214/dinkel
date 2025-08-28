package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type Properties struct{}

// Generate subclauses for Properties
func (c *Properties) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&PropertyChain{CreateNewPropertyProbability: 0.5}}
}

// TemplateString for Properties
func (c Properties) TemplateString() string {
	return "{%s}"
}

type PropertyChain struct {
	isBasecase bool

	// Probability of choosing to create a new property instead of using an existing one
	CreateNewPropertyProbability float64
}

// Generate subclauses for PropertyChain
func (c *PropertyChain) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.isBasecase = seed.RandomBoolean()

	var property translator.Clause = &ExistingProperty{}
	if seed.BooleanWithProbability(c.CreateNewPropertyProbability) {
		property = &NewProperty{}
	}

	if c.isBasecase {
		return []translator.Clause{property}
	}
	return []translator.Clause{property, &PropertyChain{CreateNewPropertyProbability: c.CreateNewPropertyProbability}}
}

// TemplateString for PropertyChain
func (c PropertyChain) TemplateString() string {
	if c.isBasecase {
		return "%s"
	}
	return "%s, %s"
}

func (c PropertyChain) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	return clause.GetSubclauseClauseCapturers()[0]
}

type OptionalProperties struct{}

// Generate subclauses for OptionalProperties
func (c *OptionalProperties) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		return []translator.Clause{&Properties{}}
	}
	return nil
}

// Property is currently unused because properties get generated using [PropertyChain].
// However, it might be useful in the future for implementation specific generation.
type Property struct{}

// Generate subclauses for Property
func (c *Property) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	switch seed.GetRandomIntn(2) {
	case 0:
		return []translator.Clause{&NewProperty{}}
	case 1:
		return []translator.Clause{&ExistingProperty{}}
	}
	return nil
}

type NewProperty struct {
	name string
}

// Generate subclauses for NewProperty
func (c *NewProperty) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.name = generateUniqueName(seed, s)
	propertyType := generatePropertyType(seed)
	property := schema.Property{
		Name: c.name,
		Type: propertyType,
	}
	s.AddProperty(property)

	return []translator.Clause{&Expression{Conf: schema.ExpressionConfig{TargetType: schema.PropertyValue, PropertyType: propertyType, MustBeNonNull: s.IsInMergeClause}}}
}

// TemplateString for NewProperty
func (c NewProperty) TemplateString() string {
	return c.name + ":%s"
}

type ExistingProperty struct {
	name  string
	value string
}

// Generate subclauses for ExistingProperty
func (c *ExistingProperty) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if len(s.Properties[schema.AnyType]) > 0 {
		availableProps := s.Properties[schema.AnyType]
		property := availableProps[seed.GetRandomIntn(len(availableProps))]
		c.name = property.Name
		targetType := property.Type
		// Set the target to a random type with some probability
		if seed.RandomBoolean() {
			targetType = schema.AnyType
			s.AddProperty(schema.Property{
				Name: c.name,
				Type: targetType,
			})
		}
		// Use the property's value with some probability
		if property.Value != "" && seed.BooleanWithProbability(0.75) {
			c.value = property.Value
			return nil
		}
		return []translator.Clause{&Expression{Conf: schema.ExpressionConfig{TargetType: schema.PropertyValue, PropertyType: targetType, MustBeNonNull: s.IsInMergeClause}}}
	}
	return []translator.Clause{&NewProperty{}}
}

// TemplateString for ExistingProperty
func (c ExistingProperty) TemplateString() string {
	if c.name == "" {
		return "%s"
	}
	if c.value == "" {
		return c.name + ":%s"
	}
	return c.name + ":" + c.value
}

type OptionalPropertyMatch struct{}

// Generate subclauses for OptionalPropertyMatch
func (c *OptionalPropertyMatch) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		return nil
	}
	return []translator.Clause{&PropertiesMatch{}}
}

// PropertiesMatch is similar to [Properties], but with different probabilities of matching new and existing properties.
type PropertiesMatch struct{}

// Generate subclauses for PropertiesMatch
func (c *PropertiesMatch) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&PropertyChain{CreateNewPropertyProbability: 0.05}}
}

// TemplateString for PropertiesMatch
func (c PropertiesMatch) TemplateString() string {
	return "{%s}"
}
