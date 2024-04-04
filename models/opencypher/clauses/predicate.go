package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

type Predicate struct {
	Conf      schema.ExpressionConfig
	oldSchema schema.Schema
	funcName  string
}

// Generate subclauses for Predicate
func (c *Predicate) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.oldSchema = *s.Copy()

	c.funcName = seed.RandomStringFromChoice("all", "any", "none", "single")
	return []translator.Clause{&PredicatePrefix{Conf: c.Conf}, &WhereClause{}}
}

// TemplateString for Predicate
func (c Predicate) TemplateString() string {
	return c.funcName + "(%s %s)"
}

func (c Predicate) ModifySchema(s *schema.Schema) {
	*s = c.oldSchema
}

// The PredicatePrefix represents the left part of the predicate which defines a new variable.
type PredicatePrefix struct {
	iteratorName string
	Conf         schema.ExpressionConfig
	iteratorConf schema.ExpressionConfig
}

// Generate subclauses for PredicatePrefix
func (c *PredicatePrefix) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.iteratorName = generateUniqueName(seed, s)
	c.iteratorConf = generateExpressionConf(seed)
	c.iteratorConf.IsList = false
	c.iteratorConf.IsConstantExpression = c.Conf.IsConstantExpression
	c.iteratorConf.CanContainAggregatingFunctions = c.Conf.CanContainAggregatingFunctions
	c.iteratorConf.MustBeNonNull = c.Conf.MustBeNonNull

	listConf := c.iteratorConf
	listConf.IsList = true
	return []translator.Clause{&Expression{Conf: listConf}}
}

// TemplateString for PredicatePrefix
func (c PredicatePrefix) TemplateString() string {
	return c.iteratorName + " IN %s"
}

func (c PredicatePrefix) ModifySchema(s *schema.Schema) {
	addVariableToSchema(s, c.iteratorName, c.iteratorConf)
}
