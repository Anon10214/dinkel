package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

type Remove struct{}

// Generate subclauses for Remove
func (c *Remove) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if config.GetConfig().OnlyVariablesAsWriteTarget &&
		len(s.StructuralVariablesByType[schema.NODE])+len(s.StructuralVariablesByType[schema.RELATIONSHIP]) == 0 {
		return []translator.Clause{&EmptyClause{}, &WriteClause{}}
	}
	return []translator.Clause{&RemoveClause{}, &OptionalWriteQuery{}}
}

// TemplateString for Remove
func (c Remove) TemplateString() string {
	return "%s %s"
}

type RemoveClause struct{}

// Generate subclauses for RemoveClause
func (c *RemoveClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&RemoveSubclause{}}
}

// TemplateString for RemoveClause
func (c RemoveClause) TemplateString() string {
	return "REMOVE %s"
}

type RemoveSubclause struct {
	isBasecase bool
}

// Generate subclauses for RemoveSubclause
func (c *RemoveSubclause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.isBasecase = seed.RandomBoolean()

	var subclause translator.Clause
	if seed.RandomBoolean() {
		subclause = &RemovePropertyExpression{}
	} else {
		subclause = &RemoveLabelExpression{}
	}

	if c.isBasecase {
		return []translator.Clause{subclause}
	}
	return []translator.Clause{subclause, &RemoveSubclause{}}
}

// TemplateString for RemoveSubclause
func (c RemoveSubclause) TemplateString() string {
	if c.isBasecase {
		return "%s"
	}
	return "%s, %s"
}

type RemovePropertyExpression struct{}

// Generate subclauses for RemovePropertyExpression
func (c *RemovePropertyExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {

	targetType := schema.NODE
	if config.GetConfig().OnlyVariablesAsWriteTarget {
		if len(s.StructuralVariablesByType[schema.NODE]) == 0 ||
			(len(s.StructuralVariablesByType[schema.RELATIONSHIP]) != 0 && seed.RandomBoolean()) {
			targetType = schema.RELATIONSHIP
		}
	} else if seed.RandomBoolean() {
		targetType = schema.RELATIONSHIP
	}

	return []translator.Clause{&WriteTarget{TargetType: targetType}, &PropertyName{}}
}

// TemplateString for RemovePropertyExpression
func (c RemovePropertyExpression) TemplateString() string {
	return "%s.%s"
}

// RemoveLabelExpression represents the expression used for stripping labels from
// a structural variable.
// If there is no available candidate, it defaults to generating a RemovePropertyExpression.
type RemoveLabelExpression struct {
	name string
}

// Generate subclauses for RemoveLabelExpression
func (c *RemoveLabelExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	availableVariables := s.StructuralVariablesByType[schema.NODE]

	if len(availableVariables) == 0 {
		return []translator.Clause{&RemovePropertyExpression{}}
	}
	c.name = availableVariables[seed.GetRandomIntn(len(availableVariables))].Name
	return []translator.Clause{&Labels{}}
}

// TemplateString for RemoveLabelExpression
func (c RemoveLabelExpression) TemplateString() string {
	if c.name == "" {
		return "%s"
	}
	return c.name + "%s"
}
