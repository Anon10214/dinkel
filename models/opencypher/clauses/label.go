package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type Labels struct {
	LabelType schema.StructuralType
}

// Generate subclauses for Labels
func (c *Labels) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	subclauses := []translator.Clause{&Label{LabelType: c.LabelType}}

	if seed.RandomBoolean() {
		// Break chain
		subclauses = append(subclauses, &EmptyClause{})
	} else {
		// Continue chain
		subclauses = append(subclauses, &Labels{LabelType: c.LabelType})
	}

	return subclauses
}

type Label struct {
	LabelType schema.StructuralType
}

// Generate subclauses for Label
func (c *Label) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&LabelName{LabelType: c.LabelType}}
}

// TemplateString for Label
func (c Label) TemplateString() string {
	return ":%s"
}

type LabelName struct {
	LabelType schema.StructuralType
}

// Generate subclauses for LabelName
func (c *LabelName) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.BooleanWithProbability(0.1) {
		return []translator.Clause{&NewLabel{LabelType: c.LabelType}}
	}
	return []translator.Clause{&ExistingLabel{LabelType: c.LabelType}}
}

type NewLabel struct {
	LabelType schema.StructuralType
	name      string
}

// Generate subclauses for NewLabel
func (c *NewLabel) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.name = generateName(seed)
	// Add the new label to the relevant slices
	if c.LabelType != schema.ANY {
		s.Labels[c.LabelType] = append(s.Labels[c.LabelType], c.name)
	}
	s.Labels[schema.ANY] = append(s.Labels[schema.ANY], c.name)
	return nil
}

// TemplateString for NewLabel
func (c NewLabel) TemplateString() string {
	return c.name
}

type ExistingLabel struct {
	LabelType schema.StructuralType
	name      string
}

// Generate subclauses for ExistingLabel
func (c *ExistingLabel) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	// Disregard label type with some probability, choosing from the pool of all labels instead
	if seed.BooleanWithProbability(0.1) {
		c.LabelType = schema.ANY
	}
	if len(s.Labels[c.LabelType]) > 0 {
		c.name = s.Labels[c.LabelType][seed.GetRandomIntn(len(s.Labels[c.LabelType]))]
		return nil
	}
	return []translator.Clause{&NewLabel{LabelType: c.LabelType}}
}

// TemplateString for ExistingLabel
func (c ExistingLabel) TemplateString() string {
	if c.name != "" {
		return c.name
	}
	return "%s"
}

type LabelMatch struct {
	LabelType schema.StructuralType
	// Whether this clause should stop recursing and generate a single label instead of combining two more subclauses
	isBasecase bool
	operator   string
	isNegated  bool
	// For the template string
	useNewSyntax bool
}

// Generate subclauses for LabelMatch
func (c *LabelMatch) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.useNewSyntax = *s.UseNewLabelMatchType && !s.IsInSubquery
	// Decide whether to negate the label (only available in the new syntax)
	if c.useNewSyntax {
		c.isNegated = seed.RandomBoolean()
	}
	// Decide if this subclause should be the base case
	if seed.BooleanWithProbability(0.65) {
		c.isBasecase = true
		if seed.BooleanWithProbability(0.9) {
			return []translator.Clause{&ExistingLabel{LabelType: c.LabelType}}
		}
		return []translator.Clause{&NewLabel{LabelType: c.LabelType}}
	}
	if c.useNewSyntax {
		c.operator = seed.RandomStringFromChoice("&", "|")
	}
	return []translator.Clause{&LabelMatch{LabelType: c.LabelType}, &LabelMatch{LabelType: c.LabelType}}
}

// TemplateString for LabelMatch
func (c LabelMatch) TemplateString() string {
	prefix := ""
	if c.isNegated {
		prefix = "!"
	}
	if c.isBasecase {
		return prefix + "%s"
	}
	if !c.useNewSyntax {
		if c.LabelType == schema.RELATIONSHIP {
			return "%s|%s"
		}
		return "%s:%s"
	}
	return prefix + "(%s" + c.operator + "%s)"
}

func (c LabelMatch) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	return &NewLabel{LabelType: c.LabelType}
}

type OptionalLabelMatch struct {
	LabelType schema.StructuralType
	// Whether a label match will be generated
	generateMatch bool
}

// Generate subclauses for OptionalLabelMatch
func (c *OptionalLabelMatch) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		return nil
	}
	c.generateMatch = true
	return []translator.Clause{&LabelMatch{LabelType: c.LabelType}}
}

// TemplateString for OptionalLabelMatch
func (c OptionalLabelMatch) TemplateString() string {
	if c.generateMatch {
		return ":%s"
	}
	return ""
}
