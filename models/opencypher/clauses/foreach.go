package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

type Foreach struct{}

// Generate subclauses for Foreach
func (c *Foreach) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&ForeachClause{}, &OptionalWriteQuery{}}
}

// TemplateString for Foreach
func (c Foreach) TemplateString() string {
	return "%s %s"
}

type ForeachClause struct {
	oldSchema schema.Schema
}

// Generate subclauses for ForeachClause
func (c *ForeachClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.oldSchema = *s

	decideOnLabelMatchType(seed, s)

	*s = *s.NewSubContext()

	return []translator.Clause{&ForeachVariable{}, &ForeachCommand{}}
}

// TemplateString for ForeachClause
func (c ForeachClause) TemplateString() string {
	return "FOREACH ( %s | %s )"
}

func (c ForeachClause) ModifySchema(s *schema.Schema) {

	c.oldSchema.UsedNames = s.UsedNames

	*s = c.oldSchema
}

type ForeachVariable struct {
	name    string
	varConf schema.ExpressionConfig
}

// Generate subclauses for ForeachVariable
func (c *ForeachVariable) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.name = generateUniqueName(seed, s)
	c.varConf = generateExpressionConf(seed)
	c.varConf.IsList = false

	listConf := c.varConf
	listConf.IsList = true

	return []translator.Clause{&Expression{Conf: listConf}}
}

// TemplateString for ForeachVariable
func (c ForeachVariable) TemplateString() string {
	return c.name + " IN %s"
}

func (c ForeachVariable) ModifySchema(s *schema.Schema) {
	addVariableToSchema(s, c.name, c.varConf)
}

type ForeachCommand struct{}

// Generate subclauses for ForeachCommand
func (c *ForeachCommand) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {

	availableCommands := []translator.Clause{
		&CreateClause{},
		&MergeClause{},
		&ForeachClause{},
	}

	nodeVariables := s.StructuralVariablesByType[schema.NODE]
	relationShipVariables := s.StructuralVariablesByType[schema.RELATIONSHIP]
	availableVariables := append(nodeVariables, relationShipVariables...)
	if len(availableVariables) != 0 {
		availableCommands = append(availableCommands, &SetClause{}, &DeleteClause{})
	}

	command := availableCommands[seed.GetRandomIntn(len(availableCommands))]
	return []translator.Clause{command, optionalClause(seed, &ForeachCommand{})}
}

// TemplateString for ForeachCommand
func (c ForeachCommand) TemplateString() string {
	return "%s %s"
}
