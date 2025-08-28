package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

type Create struct{}

// Generate subclauses for Create
func (c *Create) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&CreateClause{}, &OptionalWriteQuery{}}
}

// TemplateString for Create
func (c Create) TemplateString() string {
	return "%s %s"
}

type CreateClause struct{}

// Generate subclauses for CreateClause
func (c *CreateClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	s.JustCreatedStructuralVariables = []schema.StructuralVariable{}
	return []translator.Clause{&CreateElementChain{}}
}

func (c *CreateClause) ModifySchema(s *schema.Schema) {
	for _, variable := range s.JustCreatedStructuralVariables {
		s.AddStructuralVariable(variable)
	}
}

// TemplateString for CreateClause
func (c CreateClause) TemplateString() string {
	return "CREATE %s"
}

type CreateElementChain struct {
	isBasecase bool
}

// Generate subclauses for CreateElementChain
func (c *CreateElementChain) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	subclauses := []translator.Clause{&CreateElement{}}
	c.isBasecase = true
	if seed.RandomBoolean() {
		c.isBasecase = false
		subclauses = append(subclauses, &CreateElementChain{})
	}
	return subclauses
}

// TemplateString for CreateElementChain
func (c CreateElementChain) TemplateString() string {
	if c.isBasecase {
		return "%s"
	}
	return "%s, %s"
}

func (c CreateElementChain) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	return clause.GetSubclauseClauseCapturers()[len(clause.GetSubclauseClauseCapturers())-1]
}

type CreateElement struct {
	pathVariableName string
}

// Generate subclauses for CreateElement
func (c *CreateElement) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		c.pathVariableName = generateUniqueName(seed, s)
		s.JustCreatedStructuralVariables = append(s.JustCreatedStructuralVariables, schema.StructuralVariable{
			Name: c.pathVariableName,
			Type: schema.PATH,
		})
	}
	return []translator.Clause{&CreatePathElement{}}
}

// TemplateString for CreateElement
func (c CreateElement) TemplateString() string {
	if c.pathVariableName != "" {
		return c.pathVariableName + " = %s"
	}
	return "%s"
}

type CreatePathElement struct {
	direction relationshipDirection
	// Whether this element is part of a relationship being created
	InRelationship   bool
	relationshipName string
}

// Generate subclauses for CreatePathElement
func (c *CreatePathElement) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.direction = relationshipDirection(seed.GetRandomIntn(3) - 1)
	switch c.direction {
	case Left, Right:
		// Give the relationship a name with some probability
		if seed.RandomBoolean() {
			c.relationshipName = generateUniqueName(seed, s)
			s.JustCreatedStructuralVariables = append(s.JustCreatedStructuralVariables, schema.StructuralVariable{
				Name: c.relationshipName,
				Type: schema.RELATIONSHIP,
			})
		}
		return []translator.Clause{&CreateNode{InRelationship: true}, &Label{LabelType: schema.RELATIONSHIP}, &OptionalProperties{}, &CreatePathElement{InRelationship: true}}
	case None:
		return []translator.Clause{&CreateNode{InRelationship: c.InRelationship}}
	}
	return nil
}

// TemplateString for CreatePathElement
func (c CreatePathElement) TemplateString() string {
	switch c.direction {
	case Left:
		return "%s<-[" + c.relationshipName + "%s%s]-%s"
	case Right:
		return "%s-[" + c.relationshipName + "%s%s]->%s"
	case None:
		return "%s"
	}
	return ""
}

func (c CreatePathElement) NoStrategyReduce(clause *helperclauses.ClauseCapturer) translator.Clause {
	return clause.GetSubclauseClauseCapturers()[len(clause.GetSubclauseClauseCapturers())-1]
}

type CreateNode struct {
	// Can only return CreateExisting if set to true (cannot recreate existing node)
	InRelationship bool
}

// Generate subclauses for CreateNode
func (c *CreateNode) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if !c.InRelationship || seed.RandomBoolean() {
		return []translator.Clause{&CreateNewNode{}}
	}
	return []translator.Clause{&CreateExistingNode{}}
}

type CreateNewNode struct {
	name string
}

// Generate subclauses for CreateNewNode
func (c *CreateNewNode) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	// Name the node with some probability
	if seed.RandomBoolean() {
		c.name = generateUniqueName(seed, s)
		s.JustCreatedStructuralVariables = append(s.JustCreatedStructuralVariables, schema.StructuralVariable{
			Name: c.name,
			Type: schema.NODE,
		})
	}
	return []translator.Clause{&Labels{LabelType: schema.NODE}, &OptionalProperties{}}
}

// TemplateString for CreateNewNode
func (c CreateNewNode) TemplateString() string {
	return "(" + c.name + "%s%s)"
}

type CreateExistingNode struct {
	// Empty if no name was found
	usedName string
}

// Generate subclauses for CreateExistingNode
func (c *CreateExistingNode) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	justCreatedNodes := []schema.StructuralVariable{}
	for _, variable := range s.JustCreatedStructuralVariables {
		if variable.Type == schema.NODE {
			justCreatedNodes = append(justCreatedNodes, variable)
		}
	}
	availableNodes := append(s.StructuralVariablesByType[schema.NODE], justCreatedNodes...)
	if len(availableNodes) != 0 {
		targetNode := availableNodes[seed.GetRandomIntn(len(availableNodes))]
		if !targetNode.LikelyNull {
			c.usedName = targetNode.Name
			return nil
		}
	}
	return []translator.Clause{&CreateNode{}}
}

// TemplateString for CreateExistingNode
func (c CreateExistingNode) TemplateString() string {
	if c.usedName != "" {
		return "(" + c.usedName + ")"
	}
	return "%s"
}
