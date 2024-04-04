package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

type PropertyLiteral struct {
	Conf  schema.ExpressionConfig
	value string
}

// Generate subclauses for PropertyLiteral
func (c *PropertyLiteral) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if c.Conf.PropertyType == schema.AnyType {
		c.Conf.PropertyType = generatePropertyType(seed)
	}
	c.value = generateLiteral(seed, c.Conf.PropertyType, c.Conf.MustBeNonNull)
	return nil
}

// TemplateString for PropertyLiteral
func (c PropertyLiteral) TemplateString() string {
	return "(" + c.value + ")"
}
