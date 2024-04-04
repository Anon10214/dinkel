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
	indexName          string
}

// Generate subclauses for Index
func (c *Index) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.generateConstraint = seed.RandomBoolean()
	if c.generateConstraint {
		return []translator.Clause{&Constraint{}}
	}

	// TODO: Yeah, this is bad. Maybe just add generateName to seed and generateUniqueName to schema?
	c.indexName = seed.RandomStringFromChoice("x", "y", "z")
	s.Indexes = append(s.Indexes, c.indexName)

	switch seed.GetRandomIntn(4) {
	case 0:
		c.indexType = "RANGE"
		return []translator.Clause{&IndexOnProperties{}}
	case 1:
		c.indexType = "LOOKUP"
		return []translator.Clause{&IndexOnLabels{}}
	case 2:
		c.indexType = "TEXT"
		return []translator.Clause{&IndexOnProperty{}}
	case 3:
		c.indexType = "POINT"
		return []translator.Clause{&IndexOnProperty{}}
	}
	return nil
}

// TemplateString for Index
func (c Index) TemplateString() string {
	if c.generateConstraint {
		return "%s"
	}
	return fmt.Sprintf("CREATE %s INDEX %s IF NOT EXISTS %%s", c.indexType, c.indexName)
}

// DropIndex represents the statement dropping an existing index.
type DropIndex struct {
	useIfExist bool
	indexName  string
}

// Generate subclauses for DropIndex
func (c *DropIndex) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	// Drop a random probably non-existent index with some probability
	if seed.BooleanWithProbability(0.10) {
		c.useIfExist = true
		c.indexName = seed.RandomStringFromChoice("x", "y", "z")
		return nil
	}
	if len(s.Indexes) == 0 {
		return []translator.Clause{&Index{}}
	}
	c.indexName = seed.RandomStringFromChoice(s.Indexes...)
	c.useIfExist = seed.RandomBoolean()
	return nil
}

// TemplateString for DropIndex
func (c DropIndex) TemplateString() string {
	if c.indexName == "" {
		return "%s"
	}
	suffix := ""
	if c.useIfExist {
		suffix = " IF EXIST"
	}
	return "DROP INDEX " + c.indexName + suffix
}

// IndexOnProperties generates an index that indexes multiple properties.
// Applicable for index types: range
type IndexOnProperties struct {
	// If false, is for relationship
	isForNode bool
	varName   string
}

// Generate subclauses for IndexOnProperties
func (c *IndexOnProperties) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.isForNode = seed.RandomBoolean()
	c.varName = seed.RandomStringFromChoice("x", "y", "z")
	return []translator.Clause{&clauses.LabelName{}, &IndexOnPropertiesProperties{varName: c.varName}}
}

// TemplateString for IndexOnProperties
func (c IndexOnProperties) TemplateString() string {
	if c.isForNode {
		return fmt.Sprintf("FOR (%s:%%s) ON (%%s)", c.varName)
	}
	return fmt.Sprintf("FOR ()-[%s:%%s]-() ON (%%s)", c.varName)
}

// IndexOnPropertiesProperties represents the properties on which the IndexOnProperties generates the index
type IndexOnPropertiesProperties struct {
	hasNext bool
	varName string
}

// Generate subclauses for IndexOnPropertiesProperties
func (c *IndexOnPropertiesProperties) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	subclauses := []translator.Clause{&clauses.PropertyName{}}

	c.hasNext = seed.RandomBoolean()
	if c.hasNext {
		subclauses = append(subclauses, &IndexOnPropertiesProperties{varName: c.varName})
	}

	return subclauses
}

// TemplateString for IndexOnPropertiesProperties
func (c IndexOnPropertiesProperties) TemplateString() string {
	var suffix string
	if c.hasNext {
		suffix = ", %s"
	}
	return c.varName + ".%s" + suffix
}

// IndexOnLabels generates an index that indexes on labels.
// Applicable for index types: lookup
type IndexOnLabels struct {
	// If false, is for relationship
	isForNode bool
	varName   string
}

// Generate subclauses for IndexOnLabels
func (c *IndexOnLabels) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.isForNode = seed.RandomBoolean()
	c.varName = seed.RandomStringFromChoice("x", "y", "z")
	return nil
}

// TemplateString for IndexOnLabels
func (c IndexOnLabels) TemplateString() string {
	if c.isForNode {
		return fmt.Sprintf("FOR (%s) ON EACH labels(%s)", c.varName, c.varName)
	}
	return fmt.Sprintf("FOR ()-[%s]-() ON EACH type(%s)", c.varName, c.varName)
}

// IndexOnProperty generates an index that only indexes on a single property.
// Applicable for index types: text, point
type IndexOnProperty struct {
	// If false, is for relationship
	isForNode bool
	varName   string
}

// Generate subclauses for IndexOnProperty
func (c *IndexOnProperty) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.isForNode = seed.RandomBoolean()
	c.varName = seed.RandomStringFromChoice("x", "y", "z")
	return []translator.Clause{&clauses.LabelName{}, &clauses.PropertyName{}}
}

// TemplateString for IndexOnProperty
func (c IndexOnProperty) TemplateString() string {
	templateString := "FOR "
	if c.isForNode {
		templateString += "(" + c.varName + ":%s)"
	} else {
		templateString += "()-[" + c.varName + ":%s]-()"
	}
	return templateString + " ON (" + c.varName + ".%s)"
}
