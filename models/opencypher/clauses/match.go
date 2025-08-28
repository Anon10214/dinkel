package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type Match struct{}

// Generate subclauses for Match
func (c *Match) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&MatchClause{}, &ReadClause{}}
}

type MatchClause struct {
	isOptionalMatch bool
}

// Generate subclauses for MatchClause
func (c *MatchClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	decideOnLabelMatchType(seed, s)
	if (s.HasOptionalMatch && config.GetConfig().DisallowMatchAfterOptionalMatch) || seed.RandomBoolean() {
		c.isOptionalMatch = true
		s.HasOptionalMatch = true
	}
	return []translator.Clause{&MatchElementChain{IsOptional: c.isOptionalMatch}, &OptionalWhereClause{}}
}

// TemplateString for MatchClause
func (c MatchClause) TemplateString() string {
	if c.isOptionalMatch {
		return "OPTIONAL MATCH %s %s "
	}
	return "MATCH %s %s "
}

type MatchElementChain struct {
	IsBaseCase bool
	IsOptional bool
}

// Generate subclauses for MatchElementChain
func (c *MatchElementChain) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.BooleanWithProbability(0.9) {
		c.IsBaseCase = true
		return []translator.Clause{&PathPatternExpression{IsOptional: c.IsOptional}}
	}
	return []translator.Clause{&PathPatternExpression{IsOptional: c.IsOptional}, &MatchElementChain{IsOptional: c.IsOptional}}
}

// TemplateString for MatchElementChain
func (c MatchElementChain) TemplateString() string {
	if c.IsBaseCase {
		return "%s"
	}
	return "%s, %s"
}

func (c MatchElementChain) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	if c.IsBaseCase {
		return clause.GetSubclauseClauseCapturers()[0]
	}
	return clause.GetSubclauseClauseCapturers()[1]
}

type OptionalWhereClause struct {
	willGenerate bool
}

// Generate subclauses for OptionalWhereClause
func (c *OptionalWhereClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		return nil
	}
	c.willGenerate = true
	return []translator.Clause{&WhereClause{}}
}

type WhereClause struct{}

// Generate subclauses for WhereClause
func (c *WhereClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&WhereExpression{}}
}

// TemplateString for WhereClause
func (c WhereClause) TemplateString() string {
	return "WHERE %s"
}

type WhereExpression struct{}

// Generate subclauses for WhereExpression
func (c *WhereExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&Expression{Conf: schema.ExpressionConfig{TargetType: schema.PropertyValue, PropertyType: schema.Boolean}}}
}
