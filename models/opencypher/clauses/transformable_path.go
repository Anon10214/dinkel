package clauses

import (
	"fmt"
	"slices"

	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type TransformablePath struct {
	isChild    bool
	IsOptional bool
}

func (c *TransformablePath) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	switch seed.GetRandomIntn(2) {
	case 0:
		if c.isChild {
			return []translator.Clause{&PathPatternExpression{isChild: true, IsOptional: c.IsOptional}}
		}
		return []translator.Clause{&ReversiblePath{IsOptional: c.IsOptional}}
	case 1:
		return []translator.Clause{&CyclicPath{IsOptional: c.IsOptional}}
	}
	return nil
}

// ReversiblePath is a path that is reversed during transformation. E.g.:
//
//	(a)-[b]->(c) ==> (c)<-[b]-(a)
type ReversiblePath struct {
	direction              relationshipDirection
	templateString         string
	reversedTemplateString string
	IsOptional             bool
}

func (c *ReversiblePath) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	decideOnLabelMatchType(seed, s)

	subclauses := []translator.Clause{&MatchNode{IsOptional: c.IsOptional}}
	c.templateString = "%s"
	c.reversedTemplateString = "%s"

	for seed.RandomBoolean() {
		subclauses = append(subclauses, &MatchRelationship{IsOptional: c.IsOptional}, &MatchNode{IsOptional: c.IsOptional})

		c.direction = relationshipDirection(seed.GetRandomIntn(3) - 1)
		if c.direction == None {
			c.direction = Any
		}

		switch c.direction {
		case Left:
			c.templateString += "<-%s-%s"
			c.reversedTemplateString = "%s-%s->" + c.reversedTemplateString
		case Right:
			c.templateString += "-%s->%s"
			c.reversedTemplateString = "%s<-%s-" + c.reversedTemplateString
		case Any:
			c.templateString += "-%s-%s"
			c.reversedTemplateString = "%s-%s-" + c.reversedTemplateString
		}
	}

	return subclauses
}

// TemplateString for PathPatternExpression
func (c ReversiblePath) TemplateString() string {
	return c.templateString
}

func (c *ReversiblePath) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	reversedClauses := make([]translator.Clause, len(subclauses))
	copy(reversedClauses, subclauses)
	slices.Reverse(reversedClauses)
	return helperclauses.CreateAssembler(reversedClauses, c.reversedTemplateString)
}

// CyclicPath is a path that matches a cycle. E.g.:
//
//	(a)-->(a) ==> (a)-->(a)-->(a)
type CyclicPath struct {
	varName        string
	templateString string
	IsOptional     bool
}

func cyclicPathStep(varName string, seed *seed.Seed) string {
	direction := relationshipDirection(seed.GetRandomIntn(3) - 1)
	if direction == None {
		direction = Any
	}

	switch direction {
	case Left:
		return fmt.Sprintf("<--(%s)", varName)
	case Right:
		return fmt.Sprintf("-->(%s)", varName)
	case Any:
		return fmt.Sprintf("--(%s)", varName)
	}

	return ""
}

func (c *CyclicPath) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if c.varName == "" {
		c.varName = generateUniqueName(seed, s)
		s.JustCreatedStructuralVariables = append(s.JustCreatedStructuralVariables, schema.StructuralVariable{
			Name:       c.varName,
			Type:       schema.NODE,
			LikelyNull: c.IsOptional,
		})
	}
	subclauses := []translator.Clause{}

	if seed.RandomBoolean() {
		subclauses = append(subclauses, &PathPatternExpression{isChild: true, IsOptional: c.IsOptional})
		c.templateString += "%s<--"
	}

	c.templateString += "%s"
	subclauses = append(subclauses, helperclauses.CreateAssembler(
		[]translator.Clause{optionalClause(seed, &PropertiesMatch{})}, fmt.Sprintf("(%s%%s)", c.varName),
	))

	c.templateString += cyclicPathStep(c.varName, seed)
	for seed.RandomBoolean() {
		c.templateString += cyclicPathStep(c.varName, seed)
	}

	lastDirection := relationshipDirection(seed.GetRandomIntn(4) - 1)
	switch lastDirection {
	case Left:
		c.templateString += "<-%s-%s"
	case Right:
		c.templateString += "-%s->%s"
	case Any:
		c.templateString += "-%s-%s"
	}
	if lastDirection != None {
		subclauses = append(subclauses, optionalClause(seed, &MatchRelationship{IsOptional: c.IsOptional}), &PathPatternExpression{isChild: true, IsOptional: true})
	}

	return subclauses
}

func (c CyclicPath) TemplateString() string {
	return c.templateString
}

func (c *CyclicPath) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	newTemplateString := c.templateString + cyclicPathStep(c.varName, seed)
	for seed.RandomBoolean() {
		newTemplateString += cyclicPathStep(c.varName, seed)
	}
	return helperclauses.CreateAssembler(subclauses, newTemplateString)
}
