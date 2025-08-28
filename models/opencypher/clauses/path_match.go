package clauses

import (
	"fmt"

	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type PathPatternExpression struct {
	direction relationshipDirection
	// If this is the child of another path pattern expression
	isChild    bool
	IsOptional bool
}

// Generate subclauses for PathPatternExpression
func (c *PathPatternExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	decideOnLabelMatchType(seed, s)

	if seed.BooleanWithProbability(0.1) {
		return []translator.Clause{&TransformablePath{IsOptional: c.IsOptional, isChild: c.isChild}}
	}

	c.direction = relationshipDirection(seed.GetRandomIntn(4) - 1)
	if seed.BooleanWithProbability(0.75) {
		// Make it more likely to have direction none
		c.direction = None
	}
	switch c.direction {
	case Left, Right, Any:
		return []translator.Clause{&MatchNode{IsOptional: c.IsOptional}, &MatchRelationship{IsOptional: c.IsOptional}, &PathPatternExpression{IsOptional: c.IsOptional, isChild: true}}
	case None:
		return []translator.Clause{&MatchNode{IsOptional: c.IsOptional}}
	}
	return nil
}

// TemplateString for PathPatternExpression
func (c PathPatternExpression) TemplateString() string {
	switch c.direction {
	case Left:
		return "%s<-%s-%s"
	case Right:
		return "%s-%s->%s"
	case Any:
		return "%s-%s-%s"
	case None:
		return "%s"
	}
	return ""
}

func (c PathPatternExpression) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	return clause.GetSubclauseClauseCapturers()[0]
}

type MatchNode struct {
	IsOptional bool
}

// Generate subclauses for MatchNode
func (c *MatchNode) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{getOptionalStructureName(schema.NODE, c.IsOptional), &OptionalLabelMatch{LabelType: schema.NODE}, &OptionalPropertyMatch{}}
}

// TemplateString for MatchNode
func (c *MatchNode) TemplateString() string {
	return "(%s%s%s)"
}

type MatchRelationship struct {
	minVariableLength *int
	maxVariableLength *int
	hasStructureName  bool
	IsOptional        bool
}

// Generate subclauses for MatchRelationship
func (c *MatchRelationship) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	// Variable length relationships must not use relationship type expressions
	useVariableLength := false
	if !*s.UseNewLabelMatchType {
		if seed.BooleanWithProbability(0.05) {
			useVariableLength = true
			minVariableLength := seed.GetRandomIntn(10)
			c.minVariableLength = &minVariableLength
		}
		if seed.BooleanWithProbability(0.05) {
			useVariableLength = true
			maxVariableLength := seed.GetRandomIntn(20)
			c.maxVariableLength = &maxVariableLength
			if c.minVariableLength != nil {
				*c.maxVariableLength += *c.minVariableLength
			}
		}
	}
	structureType := schema.RELATIONSHIP
	if useVariableLength {
		structureType |= schema.StructuralType(schema.ListMask)
	}
	var structureName translator.Clause = &EmptyClause{}
	if seed.RandomBoolean() {
		c.hasStructureName = true
		structureName = &StructureName{NameType: &structureType, likelyNull: c.IsOptional}
	}
	return []translator.Clause{structureName, &OptionalLabelMatch{LabelType: schema.RELATIONSHIP}, &OptionalPropertyMatch{}}
}

// TemplateString for MatchRelationship
func (c *MatchRelationship) TemplateString() string {
	variableLength := ""
	if c.minVariableLength != nil || c.maxVariableLength != nil {
		variableLength = "*"
		if c.minVariableLength != nil {
			variableLength += fmt.Sprint(*c.minVariableLength)
		}
		variableLength += ".."
		if c.maxVariableLength != nil {
			variableLength += fmt.Sprint(*c.maxVariableLength)
		}
	}
	return "[%s%s" + variableLength + "%s]"
}
