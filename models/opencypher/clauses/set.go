package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type Set struct{}

// Generate subclauses for Set
func (c *Set) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if config.GetConfig().OnlyVariablesAsWriteTarget &&
		len(s.StructuralVariablesByType[schema.NODE])+len(s.StructuralVariablesByType[schema.RELATIONSHIP]) == 0 {
		return []translator.Clause{&EmptyClause{}, &WriteClause{}}
	}
	return []translator.Clause{&SetClause{}, &OptionalWriteQuery{}}
}

// TemplateString for Set
func (c Set) TemplateString() string {
	return "%s %s"
}

type SetClause struct{}

// Generate subclauses for SetClause
func (c *SetClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&SetExpression{}}
}

// TemplateString for SetClause
func (c SetClause) TemplateString() string {
	return "SET %s"
}

type SetExpression struct {
	isBasecase bool
}

// Generate subclauses for SetExpression
func (c *SetExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.isBasecase = seed.RandomBoolean()
	var subclause translator.Clause
	if seed.RandomBoolean() {
		subclause = &SetPropertyExpression{}
	} else {
		subclause = &SetLabelExpression{}
	}
	if c.isBasecase {
		return []translator.Clause{subclause}
	}
	return []translator.Clause{subclause, &SetExpression{}}
}

// TemplateString for SetExpression
func (c SetExpression) TemplateString() string {
	if c.isBasecase {
		return "%s"
	}
	return "%s, %s"
}

func (c SetExpression) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	return clause.GetSubclauseClauseCapturers()[len(clause.GetSubclauseClauseCapturers())-1]
}

type SetPropertyExpression struct {
	name string
	// Whether the set clause assigns a map or just a single value
	isMapAssign bool
	// If this is an addition of a map instead of just an assign
	isMapAddition bool
}

// Generate subclauses for SetPropertyExpression
func (c *SetPropertyExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {

	decideOnLabelMatchType(seed, s)

	nodeVariables := s.StructuralVariablesByType[schema.NODE]
	relationShipVariables := s.StructuralVariablesByType[schema.RELATIONSHIP]
	availableVariables := append(nodeVariables, relationShipVariables...)
	// Only allow map assign if there exists an available variable
	c.isMapAssign = len(availableVariables) > 0 && seed.RandomBoolean()

	var subclauses []translator.Clause
	if c.isMapAssign {
		c.name = availableVariables[seed.GetRandomIntn(len(availableVariables))].Name
		c.isMapAddition = seed.RandomBoolean()
		subclauses = []translator.Clause{&Properties{}}
	} else {
		targetType := schema.NODE
		if config.GetConfig().OnlyVariablesAsWriteTarget {
			nodeVars := len(s.StructuralVariablesByType[schema.NODE])
			relationshipVars := len(s.StructuralVariablesByType[schema.RELATIONSHIP])
			if nodeVars == 0 || (relationshipVars != 0 && seed.RandomBoolean()) {
				targetType = schema.RELATIONSHIP
			}
		} else if seed.RandomBoolean() {
			targetType = schema.RELATIONSHIP
		}

		subclauses = []translator.Clause{&WriteTarget{TargetType: targetType}, &PropertyName{}, &Expression{Conf: schema.ExpressionConfig{TargetType: schema.PropertyValue}}}
	}

	return subclauses
}

// TemplateString for SetPropertyExpression
func (c SetPropertyExpression) TemplateString() string {
	if c.isMapAssign {
		if c.isMapAddition {
			return c.name + " += %s"
		}
		return c.name + " = %s"
	}
	return "%s.%s = %s"
}

type SetLabelExpression struct {
	name string
}

// Generate subclauses for SetLabelExpression
func (c *SetLabelExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {

	availableVariables := s.StructuralVariablesByType[schema.NODE]

	if len(availableVariables) == 0 {
		return []translator.Clause{&SetPropertyExpression{}}
	}
	c.name = availableVariables[seed.GetRandomIntn(len(availableVariables))].Name
	return []translator.Clause{&Labels{}}
}

// TemplateString for SetLabelExpression
func (c SetLabelExpression) TemplateString() string {
	if c.name == "" {
		return "%s"
	}
	return c.name + "%s"
}
