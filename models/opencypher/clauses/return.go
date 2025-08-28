package clauses

import (
	"strings"

	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type Return struct {
	// If the return is predetermined
	isPredetermined bool
}

// Generate subclauses for Return
func (c *Return) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if s.CannotReturn {
		c.isPredetermined = true
		return []translator.Clause{&WriteClause{}}
	}
	decideOnLabelMatchType(seed, s)

	if s.MustReturn {
		c.isPredetermined = true
		return []translator.Clause{&PredeterminedReturn{}}
	}

	subclauses := []translator.Clause{optionalClause(seed, helperclauses.CreateStringer("DISTINCT"))}
	if len(s.PropertyVariablesByType[schema.AnyType])+len(s.StructuralVariablesByType[schema.ANY]) != 0 &&
		seed.RandomBoolean() && !s.IsInSubquery && !s.DisallowReturnAll {
		// Generate RETURN * if variables in scope and not in subquery
		subclauses = append(subclauses, helperclauses.CreateStringer("*"))
	} else {
		subclauses = append(subclauses, &ReturnElementChain{})
	}

	return append(subclauses, &OptionalOrderBy{}, &OptionalSkip{}, &OptionalLimit{})
}

// TemplateString for Return
func (c Return) TemplateString() string {
	if c.isPredetermined {
		return "%s"
	}
	return "RETURN %s %s %s %s %s"
}

type ReturnElementChain struct {
	isBasecase bool
}

// Generate subclauses for ReturnElementChain
func (c *ReturnElementChain) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.isBasecase = seed.RandomBoolean()

	subclauses := []translator.Clause{&ReturnElement{}}
	if !c.isBasecase {
		subclauses = append(subclauses, &ReturnElementChain{})
	}

	return subclauses
}

// TemplateString for ReturnElementChain
func (c ReturnElementChain) TemplateString() string {
	if c.isBasecase {
		return "%s"
	}
	return "%s, %s"
}

func (c ReturnElementChain) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	return clause.GetSubclauseClauseCapturers()[0]
}

type ReturnElement struct{}

// Generate subclauses for ReturnElement
func (c *ReturnElement) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&Expression{Conf: schema.ExpressionConfig{AllowMaps: true, CanContainAggregatingFunctions: true}}, &StructureName{}}
}

// TemplateString for ReturnElement
func (c ReturnElement) TemplateString() string {
	return "%s AS %s"
}

// A PredeterminedReturn has its return names and types defined in the schema
// and is only generated if the schema's MustReturn boolean is set to true
type PredeterminedReturn struct {
	aliases []string
}

// Generate subclauses for PredeterminedReturn
func (c *PredeterminedReturn) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	var subclauses []translator.Clause

	// Iterate over property variables
	for _, variable := range s.PropertyVariablesToReturn {
		subclauses = append(subclauses, &Expression{Conf: schema.ExpressionConfig{TargetType: schema.PropertyValue, PropertyType: variable.Type, CanContainAggregatingFunctions: true}})
		c.aliases = append(c.aliases, variable.Name)
	}

	// Iterate over structural variables
	for _, variable := range s.StructuralVariablesToReturn {
		c.aliases = append(c.aliases, variable.Name)
		// If no structural variable of the desired type is available, choose null
		subclauses = append(subclauses, &Expression{Conf: schema.ExpressionConfig{TargetType: schema.StructuralValue, StructuralType: variable.Type, CanContainAggregatingFunctions: true}})
	}

	return subclauses
}

// TemplateString for PredeterminedReturn
func (c PredeterminedReturn) TemplateString() string {
	var returnExpressions []string

	for _, alias := range c.aliases {
		returnExpressions = append(returnExpressions, "%s AS "+alias)
	}

	return "RETURN " + strings.Join(returnExpressions, ", ")
}

type OptionalOrderBy struct {
	isGenerated bool
}

// Generate subclauses for OptionalOrderBy
func (c *OptionalOrderBy) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		return nil
	}
	c.isGenerated = true
	return []translator.Clause{&OrderByExpressionChain{}}
}

// TemplateString for OptionalOrderBy
func (c OptionalOrderBy) TemplateString() string {
	if !c.isGenerated {
		return ""
	}
	return "ORDER BY %s"
}

type OrderByExpressionChain struct {
	isBasecase bool
}

// Generate subclauses for OrderByExpressionChain
func (c *OrderByExpressionChain) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.isBasecase = seed.RandomBoolean()

	subclauses := []translator.Clause{&OrderByExpression{}}
	if !c.isBasecase {
		subclauses = append(subclauses, &OrderByExpressionChain{})
	}

	return subclauses
}

// TemplateString for OrderByExpressionChain
func (c OrderByExpressionChain) TemplateString() string {
	if c.isBasecase {
		return "%s"
	}
	return "%s, %s"
}

type OrderByExpression struct {
	orderByType string
}

// Generate subclauses for OrderByExpression
func (c *OrderByExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	switch seed.GetRandomIntn(3) {
	case 0:
		c.orderByType = "ASC"
	case 1:
		c.orderByType = "DESC"
	}
	return []translator.Clause{&Expression{Conf: schema.ExpressionConfig{AllowMaps: true}}}
}

// TemplateString for OrderByExpression
func (c OrderByExpression) TemplateString() string {
	return "%s " + c.orderByType
}

type OptionalLimit struct {
	willGenerate bool
}

// Generate subclauses for OptionalLimit
func (c *OptionalLimit) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		c.willGenerate = true
		return []translator.Clause{&Expression{Conf: schema.ExpressionConfig{TargetType: schema.PropertyValue, PropertyType: schema.PositiveInteger, MustBeNonNull: true, IsConstantExpression: true}}}
	}
	return nil
}

// TemplateString for OptionalLimit
func (c OptionalLimit) TemplateString() string {
	if c.willGenerate {
		return "LIMIT %s"
	}
	return ""
}

type OptionalSkip struct {
	willGenerate bool
}

// Generate subclauses for OptionalSkip
func (c *OptionalSkip) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		c.willGenerate = true
		return []translator.Clause{&Expression{Conf: schema.ExpressionConfig{TargetType: schema.PropertyValue, PropertyType: schema.PositiveInteger, MustBeNonNull: true, IsConstantExpression: true}}}
	}
	return nil
}

// TemplateString for OptionalSkip
func (c OptionalSkip) TemplateString() string {
	if c.willGenerate {
		return "SKIP %s"
	}
	return ""
}
