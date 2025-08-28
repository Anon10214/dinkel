package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

type Unwind struct{}

// Generate subclauses for Unwind
func (c *Unwind) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&UnwindClause{}, &ReadClause{}}
}

// TemplateString for Unwind
func (c Unwind) TemplateString() string {
	return "%s %s"
}

// TODO: Add more deeply nested lists?

type UnwindClause struct {
	name           string
	VariableConfig schema.ExpressionConfig
}

// Generate subclauses for UnwindClause
func (c *UnwindClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.name = generateUniqueName(seed, s)

	c.VariableConfig = generateExpressionConf(seed)
	c.VariableConfig.IsList = true

	return []translator.Clause{&Expression{Conf: c.VariableConfig}}
}

// TemplateString for UnwindClause
func (c UnwindClause) TemplateString() string {
	return "UNWIND %s AS " + c.name
}

func (c UnwindClause) ModifySchema(s *schema.Schema) {
	c.VariableConfig.IsList = false
	addVariableToSchema(s, c.name, c.VariableConfig)
}
