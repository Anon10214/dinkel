package clauses

import (
	"fmt"

	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

// Index is the Neo4j implementation for creating or dropping a database index
// as well as for creating constraints on properties.
type Index struct {
	generateConstraint bool
	indexType          string
}

// Generate subclauses for Index
func (c *Index) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.generateConstraint = seed.RandomBoolean()
	if c.generateConstraint {
		return []translator.Clause{&Constraint{}}
	}

	// Decide if it is a label or property index
	if seed.RandomBoolean() {
		// Decide if it is an index on edges or not
		if seed.RandomBoolean() {
			c.indexType = " EDGE"
		}
		return []translator.Clause{&IndexOnLabel{}}
	}
	return []translator.Clause{&IndexOnProperty{}}
}

// TemplateString for Index
func (c Index) TemplateString() string {
	if c.generateConstraint {
		return "%s"
	}
	return fmt.Sprintf("CREATE%s INDEX ON %%s", c.indexType)
}

// DropIndex represents the statement dropping an existing index.
type DropIndex struct{}

// Generate subclauses for DropIndex
func (c *DropIndex) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	// TODO: Implement
	return []translator.Clause{&Index{}}
}

// IndexOnLabel generates an index that indexes on labels.
// Applicable for index types: lookup
type IndexOnLabel struct{}

// Generate subclauses for IndexOnLabels
func (c *IndexOnLabel) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&clauses.Label{}}
}

// IndexOnProperty generates an index that only indexes on a single property.
// Applicable for index types: text, point
type IndexOnProperty struct{}

// Generate subclauses for IndexOnProperty
func (c *IndexOnProperty) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&clauses.Label{}, &clauses.PropertyName{}}
}

// TemplateString for IndexOnProperty
func (c IndexOnProperty) TemplateString() string {
	return "%s(%s)"
}
