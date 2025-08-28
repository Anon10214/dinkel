package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type Union struct {
	isUnionAll bool
}

// Generate subclauses for Union
func (c *Union) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	// Define required variables to return if not yet defined
	if !s.MustReturn {
		populateVariablesToReturn(seed, s)
	}
	if s.IsUnionAll == nil {
		isUnionAll := new(bool)
		if seed.RandomBoolean() {
			*isUnionAll = true
		}
		s.IsUnionAll = isUnionAll
	}
	c.isUnionAll = *s.IsUnionAll
	return []translator.Clause{&ReadClause{}, &UnionClause{}}
}

// TemplateString for Union
func (c Union) TemplateString() string {
	if c.isUnionAll {
		return "%s UNION ALL %s"
	}
	return "%s UNION %s"
}

func (c Union) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	return clause.GetSubclauseClauseCapturers()[0]
}

type UnionClause struct {
	oldSchema schema.Schema
}

// Generate subclauses for UnionClause
func (c *UnionClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.oldSchema = *s

	newSchema := c.oldSchema.NewContext()

	newSchema.IsUnionAll = c.oldSchema.IsUnionAll

	newSchema.MustReturn = true
	newSchema.PropertyVariablesToReturn = c.oldSchema.PropertyVariablesToReturn
	newSchema.StructuralVariablesToReturn = c.oldSchema.StructuralVariablesToReturn

	*s = *newSchema
	return []translator.Clause{&ReadClause{}}
}

func (c UnionClause) ModifySchema(s *schema.Schema) {
	oldSchema := s
	*oldSchema = c.oldSchema
}
