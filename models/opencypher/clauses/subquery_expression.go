package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

type Exists struct{}

// Generate subclauses for Exists
func (c *Exists) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		return []translator.Clause{&SimpleSubqueryExpressionBody{}}
	}
	return []translator.Clause{&SubqueryExpression{}}
}

// TemplateString for Exists
func (c Exists) TemplateString() string {
	return "EXISTS { %s }"
}

type Count struct{}

// Generate subclauses for Count
func (c *Count) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		return []translator.Clause{&SimpleSubqueryExpressionBody{}}
	}
	return []translator.Clause{&SubqueryExpression{}}
}

// TemplateString for Count
func (c Count) TemplateString() string {
	return "COUNT { %s }"
}

type Collect struct {
	Conf schema.ExpressionConfig
}

// Generate subclauses for Collect
func (c *Collect) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.Conf.IsList = false
	columnName := generateUniqueName(seed, s)
	return []translator.Clause{&SubqueryExpression{ColumnNameToReturn: columnName, ColumnExpressionToReturn: c.Conf}}
}

// TemplateString for Collect
func (c Collect) TemplateString() string {
	return "COLLECT { %s }"
}

// Only has a path pattern and an optional WHERE clause
type SimpleSubqueryExpressionBody struct {
	oldSchema schema.Schema
}

// Generate subclauses for SimpleSubqueryExpressionBody
func (c *SimpleSubqueryExpressionBody) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.oldSchema = *s

	decideOnLabelMatchType(seed, s)

	*s = *s.NewSubContext()
	return []translator.Clause{&PathPatternExpression{}, optionalClause(seed, &WhereClause{})}
}

func (c SimpleSubqueryExpressionBody) ModifySchema(s *schema.Schema) {

	c.oldSchema.UsedNames = s.UsedNames

	*s = c.oldSchema
}

type SubqueryExpression struct {
	ColumnNameToReturn       string
	ColumnExpressionToReturn schema.ExpressionConfig
	oldSchema                schema.Schema
}

// Generate subclauses for SubqueryExpression
func (c *SubqueryExpression) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.oldSchema = *s

	decideOnLabelMatchType(seed, s)

	*s = *s.NewSubContext()
	// Generate return columns with some probability if none set
	if c.ColumnNameToReturn == "" && seed.RandomBoolean() {
		s.MustReturn = true
		populateVariablesToReturn(seed, s)
	} else if c.ColumnNameToReturn != "" {
		s.MustReturn = true
		if c.ColumnExpressionToReturn.TargetType == schema.PropertyValue {
			s.StructuralVariablesToReturn = []schema.StructuralVariable{
				{
					Name: c.ColumnNameToReturn,
					Type: c.ColumnExpressionToReturn.StructuralType,
				},
			}
		} else {
			s.PropertyVariablesToReturn = []schema.PropertyVariable{
				{
					Name: c.ColumnNameToReturn,
					Type: c.ColumnExpressionToReturn.PropertyType,
				},
			}
		}
	}

	var isUnionAll *bool
	if seed.RandomBoolean() {
		isUnionAllBool := seed.RandomBoolean()
		isUnionAll = &isUnionAllBool
	}
	return []translator.Clause{&SubqueryExpressionBody{IsUnionAll: isUnionAll}}
}

func (c SubqueryExpression) ModifySchema(s *schema.Schema) {

	c.oldSchema.UsedNames = s.UsedNames

	*s = c.oldSchema
}

type SubqueryExpressionBody struct {
	// If unset, no union. If set and true, UNION ALL, if set and false, UNION
	IsUnionAll *bool
	// If this body generates a UNION
	hasUnion   bool
	MustReturn bool
}

// Generate subclauses for SubqueryExpressionBody
func (c *SubqueryExpressionBody) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {

	var nextClause translator.Clause
	if c.IsUnionAll == nil || seed.RandomBoolean() {
		nextClause = &EmptyClause{}
	} else {
		c.hasUnion = true
		// If this is the first part in the chain
		if !c.MustReturn {
			populateVariablesToReturn(seed, s)
		}
		c.MustReturn = true
		nextClause = &SubqueryExpressionBodyPart{MustReturn: c.MustReturn}
	}
	return []translator.Clause{&SubqueryExpressionBodyPart{MustReturn: c.MustReturn}, nextClause}
}

// TemplateString for SubqueryExpressionBody
func (c SubqueryExpressionBody) TemplateString() string {
	suffix := "%s"
	if c.hasUnion {
		if *c.IsUnionAll {
			suffix = "UNION ALL " + suffix
		} else {
			suffix = "UNION " + suffix
		}
	}
	return "%s " + suffix
}

type SubqueryExpressionBodyPart struct {
	oldSchema  schema.Schema
	MustReturn bool
}

// Generate subclauses for SubqueryExpressionBodyPart
func (c *SubqueryExpressionBodyPart) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.oldSchema = *s

	*s = *s.NewSubContext()

	s.MustReturn = c.oldSchema.MustReturn
	s.PropertyVariablesToReturn = c.oldSchema.PropertyVariablesToReturn
	s.StructuralVariablesToReturn = c.oldSchema.StructuralVariablesToReturn

	var whereClause translator.Clause = optionalClause(seed, &WhereClause{})

	var lastSubclause translator.Clause = &EmptyClause{}
	if s.MustReturn || c.MustReturn {
		lastSubclause = &Return{}
	} else {
		// Query cannot conclude with just a MATCH, has to be either RETURN or WHERE
		whereClause = &WhereClause{}
	}
	return []translator.Clause{optionalClause(seed, &WithClause{}), &PathPatternExpression{}, whereClause, lastSubclause}
}

// TemplateString for SubqueryExpressionBodyPart
func (c SubqueryExpressionBodyPart) TemplateString() string {
	return "%s MATCH %s %s %s"
}

func (c SubqueryExpressionBodyPart) ModifySchema(s *schema.Schema) {

	c.oldSchema.UsedNames = s.UsedNames

	*s = c.oldSchema
}
