package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type CaseExpression struct {
	Conf schema.ExpressionConfig
}

// Generate subclauses for CaseExpression
func (c *CaseExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	// Set getsDeleted to false since we cannot know which expression it will evaluate to
	c.Conf.GetsDeleted = false
	decideOnLabelMatchType(seed, s)
	if seed.RandomBoolean() {
		return []translator.Clause{&SimpleCaseExpression{Conf: c.Conf}}
	}
	return []translator.Clause{&GenericCaseExpression{Conf: c.Conf}}
}

type SimpleCaseExpression struct {
	Conf schema.ExpressionConfig
}

// Generate subclauses for SimpleCaseExpression
func (c *SimpleCaseExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	var elseExpression translator.Clause
	if !c.Conf.MustBeNonNull {
		elseExpression = optionalClause(seed, &CaseExpressionElse{Conf: c.Conf})
	} else {
		elseExpression = &CaseExpressionElse{Conf: c.Conf}
	}
	return []translator.Clause{&Expression{Conf: c.Conf}, &CaseExpressionWhen{Conf: c.Conf, IsGeneric: false}, elseExpression}
}

// TemplateString for SimpleCaseExpression
func (c SimpleCaseExpression) TemplateString() string {
	return "CASE %s %s %s END"
}

type GenericCaseExpression struct {
	Conf schema.ExpressionConfig
}

// Generate subclauses for GenericCaseExpression
func (c *GenericCaseExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	var elseExpression translator.Clause
	if !c.Conf.MustBeNonNull {
		elseExpression = optionalClause(seed, &CaseExpressionElse{Conf: c.Conf})
	} else {
		elseExpression = &CaseExpressionElse{Conf: c.Conf}
	}
	return []translator.Clause{&CaseExpressionWhen{Conf: c.Conf, IsGeneric: true}, elseExpression}
}

// TemplateString for GenericCaseExpression
func (c GenericCaseExpression) TemplateString() string {
	return "CASE %s %s END"
}

type CaseExpressionWhen struct {
	Conf schema.ExpressionConfig
	// Whether this clause is part of a generic or simple case expression
	IsGeneric bool
}

// Generate subclauses for CaseExpressionWhen
func (c *CaseExpressionWhen) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	var whenClause translator.Clause
	whenConf := schema.ExpressionConfig{
		TargetType:           schema.PropertyValue,
		IsConstantExpression: c.Conf.IsConstantExpression,
	}
	if c.IsGeneric {
		whenConf.PropertyType = schema.Boolean
		whenClause = &Expression{whenConf}
	} else {
		whenConf.PropertyType = schema.AnyType
		whenClause = &Expression{whenConf}
	}
	return []translator.Clause{whenClause, &Expression{Conf: c.Conf}, optionalClause(seed, &CaseExpressionWhen{Conf: c.Conf, IsGeneric: c.IsGeneric})}
}

// TemplateString for CaseExpressionWhen
func (c CaseExpressionWhen) TemplateString() string {
	return "WHEN %s THEN %s %s"
}

func (c CaseExpressionWhen) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	return clause.GetSubclauseClauseCapturers()[len(clause.GetSubclauseClauseCapturers())-1]
}

type CaseExpressionElse struct {
	Conf schema.ExpressionConfig
}

// Generate subclauses for CaseExpressionElse
func (c *CaseExpressionElse) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&Expression{Conf: c.Conf}}
}

// TemplateString for CaseExpressionElse
func (c *CaseExpressionElse) TemplateString() string {
	return "ELSE %s"
}
