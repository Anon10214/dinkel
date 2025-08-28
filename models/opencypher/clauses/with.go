package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type With struct {
	SimpleWithClause bool
}

// Generate subclauses for With
func (c *With) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if c.SimpleWithClause {
		return []translator.Clause{&WithClause{}, &ReadClause{}}
	}
	return []translator.Clause{&WithClause{}, &OptionalOrderBy{}, &OptionalSkip{}, &OptionalLimit{}, &ReadClause{}}
}

// TemplateString for With
func (c With) TemplateString() string {
	if c.SimpleWithClause {
		return "%s %s"
	}
	return "%s %s %s %s %s"
}

func (c With) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	return clause.GetSubclauseClauseCapturers()[len(clause.GetSubclauseClauseCapturers())-1]
}

type WithClause struct {
	IsIncludeAll bool
}

// Generate subclauses for WithClause
func (c *WithClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.IsIncludeAll = seed.RandomBoolean()
	if config.GetConfig().AsteriskNeedsTargets {
		if len(s.PropertyVariablesByName)+len(s.StructuralVariablesByName) == 0 {
			c.IsIncludeAll = false
		}
	}
	if c.IsIncludeAll {
		return []translator.Clause{optionalClause(seed, helperclauses.CreateStringer("DISTINCT")), helperclauses.CreateStringer("*")}
	}
	return []translator.Clause{optionalClause(seed, helperclauses.CreateStringer("DISTINCT")), &WithElementChain{}}
}

// TemplateString for WithClause
func (c WithClause) TemplateString() string {
	return "WITH %s %s"
}

type WithElementChain struct {
	isBasecase bool

	elementName   string
	elementConfig schema.ExpressionConfig
}

// Generate subclauses for WithElementChain
func (c *WithElementChain) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.isBasecase = seed.RandomBoolean()

	expression := generateExpressionConf(seed)
	expression.CanContainAggregatingFunctions = true
	name := generateUniqueName(seed, s)

	subclauses := []translator.Clause{&WithElement{Name: name, Conf: expression}}
	if !c.isBasecase {
		subclauses = append(subclauses, &WithElementChain{})
	}
	return subclauses
}

// TemplateString for WithElementChain
func (c WithElementChain) TemplateString() string {
	if c.isBasecase {
		return "%s"
	}
	return "%s, %s"
}

func (c WithElementChain) ModifySchema(s *schema.Schema) {

	s.ResetContext()

	// Add the variables defined in WITH to the schema
	addVariableToSchema(s, c.elementName, c.elementConfig)
}

func (c WithElementChain) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	return clause.GetSubclauseClauseCapturers()[0]
}

type WithElement struct {
	Name string
	Conf schema.ExpressionConfig
}

// Generate subclauses for WithElement
func (c *WithElement) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&Expression{Conf: c.Conf}}
}

// TemplateString for WithElement
func (c WithElement) TemplateString() string {
	return "%s AS " + c.Name
}
