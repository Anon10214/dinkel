package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

/* ------------------------------------------------
 * --------------- Structure Names ----------------
 * ------------------------------------------------
 */

func getOptionalStructureName(nameType schema.StructuralType, likelyNull bool) *OptionalStructureName {
	return &OptionalStructureName{NameType: &nameType, likelyNull: likelyNull}
}

type OptionalStructureName struct {
	NameType   *schema.StructuralType
	likelyNull bool
}

// Generate subclauses for OptionalStructureName
func (c *OptionalStructureName) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		return nil
	}
	return []translator.Clause{&StructureName{NameType: c.NameType, likelyNull: c.likelyNull}}
}

type StructureName struct {
	name       string
	NameType   *schema.StructuralType
	likelyNull bool
}

// Generate subclauses for StructureName
func (c *StructureName) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if c.NameType == nil {
		c.name = generateUniqueName(seed, s)
	} else {
		c.name = generateStructureName(seed, s, *c.NameType, c.likelyNull)
	}
	return nil
}

// TemplateString for StructureName
func (c StructureName) TemplateString() string {
	return c.name
}

type PropertyName struct {
	name           string
	NameType       *schema.PropertyType
	UseExstingName bool // Used by the prometheus exporter for tracking dependencies
}

// Generate subclauses for PropertyName
func (c *PropertyName) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if c.NameType == nil {
		usedType := schema.AnyType
		c.NameType = &usedType
	}

	if len(s.Properties[schema.AnyType]) > 0 && seed.BooleanWithProbability(0.9) {
		availableProps := s.Properties[schema.AnyType]
		property := availableProps[seed.GetRandomIntn(len(availableProps))]
		c.name = property.Name
		c.UseExstingName = true
		s.AddProperty(schema.Property{
			Name: c.name,
			Type: *c.NameType,
		})
	} else {
		c.name = generatePropertyName(seed, s, *c.NameType)
	}
	return nil
}

// TemplateString for PropertyName
func (c PropertyName) TemplateString() string {
	return c.name
}
