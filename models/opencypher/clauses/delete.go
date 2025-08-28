package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

type Delete struct{}

// Generate subclauses for Delete
func (c *Delete) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if config.GetConfig().OnlyVariablesAsWriteTarget &&
		len(s.StructuralVariablesByType[schema.NODE])+len(s.StructuralVariablesByType[schema.RELATIONSHIP]) == 0 {
		return []translator.Clause{&EmptyClause{}, &WriteClause{}}
	}
	return []translator.Clause{&DeleteClause{}, &OptionalWriteQuery{}}
}

// TemplateString for Delete
func (c Delete) TemplateString() string {
	return "%s %s"
}

type DeleteClause struct {
	useDetach bool
}

// Generate subclauses for DeleteClause
func (c *DeleteClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	relationshipVars := len(s.StructuralVariablesByType[schema.RELATIONSHIP])

	if config.GetConfig().OnlyVariablesAsWriteTarget {
		if relationshipVars == 0 || seed.RandomBoolean() {
			c.useDetach = true
		}
	} else if seed.RandomBoolean() {
		c.useDetach = true
	}

	return []translator.Clause{&DeleteElementChain{UseDetach: c.useDetach}}
}

// TemplateString for DeleteClause
func (c DeleteClause) TemplateString() string {
	prefix := ""
	if c.useDetach {
		prefix = "DETACH "
	}
	return prefix + "DELETE %s"
}

type DeleteElementChain struct {
	// If this is a DETACH DELETE clause
	UseDetach bool
}

// Generate subclauses for DeleteElementChain
func (c *DeleteElementChain) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	relationshipVars := len(s.StructuralVariablesByType[schema.RELATIONSHIP])

	// Delete only relationships if a DELETE clause is generated
	targetType := schema.NODE
	if !c.UseDetach {
		targetType = schema.RELATIONSHIP
	}

	if config.GetConfig().OnlyVariablesAsWriteTarget {
		if relationshipVars != 0 {
			targetType = schema.RELATIONSHIP
		}
	} else if seed.RandomBoolean() {
		targetType = schema.RELATIONSHIP
	}

	return []translator.Clause{&WriteTarget{TargetType: targetType, GetsDeleted: true}}
}
