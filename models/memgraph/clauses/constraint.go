package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Constraint is a clause generating a Constraint statement, ensuring uniqueness
// among properties for given relationship types or node labels.
type Constraint struct{}

// Generate subclauses for Constraint
func (c *Constraint) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&Index{}}
	if seed.RandomBoolean() {
		return []translator.Clause{&NodeConstraint{}}
	}
	return []translator.Clause{&RelationshipConstraint{}}
}

// NodeConstraint creates a UNIQUE constraint on nodes.
type NodeConstraint struct{}

// Generate subclauses for NodeConstraint
func (c *NodeConstraint) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	varName := "n"
	return []translator.Clause{helperclauses.CreateStringer(varName), &clauses.Label{}, &ConstraintPropertyChain{VarName: varName}}
}

// TemplateString for NodeConstraint
func (c *NodeConstraint) TemplateString() string {
	return "CREATE CONSTRAINT IF NOT EXISTS FOR (%s%s) REQUIRE (%s) IS UNIQUE"
}

// RelationshipConstraint creates a UNIQUE constraint on relationships.
type RelationshipConstraint struct{}

// Generate subclauses for RelationshipConstraint
func (c *RelationshipConstraint) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	varName := "n"
	return []translator.Clause{helperclauses.CreateStringer(varName), &clauses.Label{}, &ConstraintPropertyChain{VarName: varName}}
}

// TemplateString for RelationshipConstraint
func (c *RelationshipConstraint) TemplateString() string {
	return "CREATE CONSTRAINT IF NOT EXISTS FOR ()-[%s%s]-() REQUIRE (%s) IS UNIQUE"
}

// ConstraintPropertyChain represents a chain of properties over which the constraints are set.
type ConstraintPropertyChain struct {
	VarName    string
	isBasecase bool
}

// Generate subclauses for ConstraintPropertyChain
func (c *ConstraintPropertyChain) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.isBasecase = seed.RandomBoolean()
	if c.isBasecase {
		return []translator.Clause{&ConstraintProperty{VarName: c.VarName}}
	}
	return []translator.Clause{&ConstraintPropertyChain{VarName: c.VarName}, &ConstraintProperty{VarName: c.VarName}}
}

// TemplateString for ConstraintPropertyChain
func (c ConstraintPropertyChain) TemplateString() string {
	if c.isBasecase {
		return "%s"
	}
	return "%s, %s"
}

// A ConstraintProperty is a single property over which the constraint is set.
type ConstraintProperty struct {
	VarName string
}

// Generate subclauses for ConstraintProperty
func (c *ConstraintProperty) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&clauses.PropertyName{}}
}

// TemplateString for ConstraintProperty
func (c ConstraintProperty) TemplateString() string {
	return c.VarName + ".%s"
}
