package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

type Merge struct{}

// Generate subclauses for Merge
func (c *Merge) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&MergeClause{}, &OptionalWriteQuery{}}
}

// TemplateString for Merge
func (c *Merge) TemplateString() string {
	return "%s %s"
}

type MergeClause struct {
	hasOnCreate                            bool
	hasOnMatch                             bool
	oldAllowOnlyNonNullPropertyExpressions bool
}

// Generate subclauses for MergeClause
func (c *MergeClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {

	c.oldAllowOnlyNonNullPropertyExpressions = s.IsInMergeClause
	s.IsInMergeClause = true

	nodeVariables := s.StructuralVariablesByType[schema.NODE]
	relationShipVariables := s.StructuralVariablesByType[schema.RELATIONSHIP]
	availableVariables := append(nodeVariables, relationShipVariables...)
	// Can generate expression if the config allows for non-variables as write targets, or else if variables are available
	canGenerateSetExpression := !config.GetConfig().OnlyVariablesAsWriteTarget || len(availableVariables) != 0

	subclauses := []translator.Clause{&CreateElement{}}

	if canGenerateSetExpression && seed.RandomBoolean() {
		c.hasOnCreate = true
		subclauses = append(subclauses, &SetExpression{})
	} else {
		subclauses = append(subclauses, &EmptyClause{})
	}

	if canGenerateSetExpression && seed.RandomBoolean() {
		c.hasOnMatch = true
		subclauses = append(subclauses, &SetExpression{})
	} else {
		subclauses = append(subclauses, &EmptyClause{})
	}
	return subclauses
}

// TemplateString for MergeClause
func (c MergeClause) TemplateString() string {
	templateString := "MERGE %s"
	if c.hasOnCreate {
		templateString += " ON CREATE SET"
	}
	templateString += " %s"
	if c.hasOnMatch {
		templateString += " ON MATCH SET"
	}
	templateString += " %s"
	return templateString
}

func (c MergeClause) ModifySchema(s *schema.Schema) {
	s.IsInMergeClause = c.oldAllowOnlyNonNullPropertyExpressions
}
