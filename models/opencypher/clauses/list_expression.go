package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type ListExpression struct {
	Conf   schema.ExpressionConfig
	isNull bool
}

// Generate subclauses for ListExpression
func (c *ListExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.BooleanWithProbability(0.1) && !c.Conf.MustBeNonNull {
		c.isNull = true
		return nil
	}

	// Generate an expression consisting of multiple list expressions with smaller probability
	if seed.BooleanWithProbability(0.2) {
		switch seed.GetRandomIntn(2) {
		case 0:
			return []translator.Clause{&ListComprehension{Conf: c.Conf}}
		case 1:
			return []translator.Clause{&OperatorApplicationExpression{Conf: c.Conf}}
		}
	}

	switch seed.GetRandomIntn(3) {
	case 0:
		return []translator.Clause{&VariableExpression{Conf: c.Conf}}
	case 1:
		return []translator.Clause{&FunctionApplicationExpression{Conf: c.Conf}}
	}
	return []translator.Clause{&ListLiteral{Conf: c.Conf}}
}

// TemplateString for ListExpression
func (c ListExpression) TemplateString() string {
	if c.isNull {
		return "null"
	}
	return "%s"
}

type ListLiteral struct {
	Conf schema.ExpressionConfig
}

// Generate subclauses for ListLiteral
func (c *ListLiteral) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		return []translator.Clause{&helperclauses.EmptyClause{}}
	}
	c.Conf.IsList = false
	return []translator.Clause{&ListLiteralItem{Conf: c.Conf}}
}

// TemplateString for ListLiteral
func (c ListLiteral) TemplateString() string {
	return "[%s]"
}

type ListLiteralItem struct {
	Conf       schema.ExpressionConfig
	isBasecase bool
}

// Generate subclauses for ListLiteralItem
func (c *ListLiteralItem) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	subclauses := []translator.Clause{&Expression{Conf: c.Conf}}
	if seed.RandomBoolean() {
		c.isBasecase = true
		return subclauses
	}
	return append(subclauses, &ListLiteralItem{Conf: c.Conf})
}

// TemplateString for ListLiteralItem
func (c ListLiteralItem) TemplateString() string {
	if c.isBasecase {
		return "%s"
	}
	return "%s, %s"
}

type ListComprehension struct {
	Conf      schema.ExpressionConfig
	oldSchema schema.Schema
}

// Generate subclauses for ListComprehension
func (c *ListComprehension) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.oldSchema = *s.Copy()
	suffixConf := c.Conf
	suffixConf.IsList = false
	suffixConf.CanContainAggregatingFunctions = false
	return []translator.Clause{&ListComprehensionPrefix{Conf: c.Conf}, &ListComprehensionSuffix{Conf: suffixConf}}
}

// TemplateString for ListComprehension
func (c ListComprehension) TemplateString() string {
	return "[%s %s]"
}

func (c ListComprehension) ModifySchema(s *schema.Schema) {
	*s = c.oldSchema
}

type ListComprehensionPrefix struct {
	Conf     schema.ExpressionConfig
	iterator string
}

// Generate subclauses for ListComprehensionPrefix
func (c *ListComprehensionPrefix) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.iterator = generateUniqueName(seed, s)
	return []translator.Clause{&Expression{Conf: c.Conf}}
}

// TemplateString for ListComprehensionPrefix
func (c ListComprehensionPrefix) TemplateString() string {
	return c.iterator + " IN %s"
}

func (c ListComprehensionPrefix) ModifySchema(s *schema.Schema) {
	c.Conf.IsList = false
	addVariableToSchema(s, c.iterator, c.Conf)
}

type ListComprehensionSuffix struct {
	Conf schema.ExpressionConfig
}

// Generate subclauses for ListComprehensionSuffix
func (c *ListComprehensionSuffix) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&OptionalWhereClause{}, &Expression{Conf: c.Conf}}
}

// TemplateString for ListComprehensionSuffix
func (c ListComprehensionSuffix) TemplateString() string {
	return "%s | %s"
}
