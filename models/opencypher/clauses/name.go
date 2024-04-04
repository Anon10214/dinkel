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
	if seed.GetRandomIntn(2) == 0 {
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

// TODO: Let PropertyName reuse names from the schema - will have to update full prometheus exporter once implemented
type PropertyName struct {
	name     string
	NameType *schema.PropertyType
}

// Generate subclauses for PropertyName
func (c *PropertyName) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if c.NameType == nil {
		c.name = generateUniqueName(seed, s)
	} else {
		c.name = generatePropertyName(seed, s, *c.NameType)
	}
	return nil
}

// TemplateString for PropertyName
func (c PropertyName) TemplateString() string {
	return c.name
}
