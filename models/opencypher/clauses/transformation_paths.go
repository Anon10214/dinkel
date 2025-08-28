package clauses

import (
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
	if c.isChild {
		return []translator.Clause{&PathPatternExpression{isChild: true, IsOptional: c.IsOptional}}
	}
	return []translator.Clause{&ReversiblePath{IsOptional: c.IsOptional}}
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
	return helperclauses.CreateAssembler(c.reversedTemplateString, reversedClauses...)
}
