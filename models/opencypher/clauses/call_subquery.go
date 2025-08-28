package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

// A CallSubquery clause must take into account the following three restrictions:
//  1. A subquery can only refer to variables from the enclosing query if they are explicitly imported.
//  2. A subquery cannot return variables with the same names as variables in the enclosing query.
//  3. All variables that are returned from a subquery are afterwards available in the enclosing query.
type CallSubquery struct{}

// Generate subclauses for CallSubquery
func (c *CallSubquery) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return []translator.Clause{&CallSubqueryClause{}, &ReadClause{}}
}

// TemplateString for CallSubquery
func (c CallSubquery) TemplateString() string {
	return "%s %s"
}

type CallSubqueryClause struct {
	oldSchema schema.Schema
	// Start the call subquery with a WITH *
	includeAll bool
}

// Generate subclauses for CallSubqueryClause
func (c *CallSubqueryClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.oldSchema = *s

	*s = *c.oldSchema.NewContext()
	s.IsInSubquery = true

	// Force the subclause to return
	if seed.RandomBoolean() {
		populateVariablesToReturn(seed, s)
	}

	// Decide which variables to include in the subquery
	var variablesToInclude []string
	// Include all variables with some probability
	c.includeAll = seed.BooleanWithProbability(0.1)
	if config.GetConfig().AsteriskNeedsTargets {
		// Make sure we can actually generate a WITH * here
		if len(s.PropertyVariablesByName)+len(s.StructuralVariablesByName) == 0 {
			c.includeAll = false
		}
	}
	for _, variable := range c.oldSchema.PropertyVariablesByName {
		if c.includeAll || seed.BooleanWithProbability(0.1) {
			s.AddPropertyVariable(variable)
			variablesToInclude = append(variablesToInclude, variable.Name)
		}
	}
	for _, variable := range c.oldSchema.StructuralVariablesByName {
		if c.includeAll || seed.BooleanWithProbability(0.1) {
			s.AddStructuralVariable(variable)
			variablesToInclude = append(variablesToInclude, variable.Name)
		}
	}

	// Succeeding WITH clause must be simple if no variables are being included
	simpleWithClause := len(variablesToInclude) == 0

	return []translator.Clause{&CallSubqueryWith{VariablesToInclude: variablesToInclude, IsIncludeAll: c.includeAll}, &ReadClause{SimpleWithClause: simpleWithClause}}
}

// TemplateString for CallSubqueryClause
func (c CallSubqueryClause) TemplateString() string {
	return "CALL { %s %s }"
}

func (c CallSubqueryClause) ModifySchema(s *schema.Schema) {
	oldSchema := s

	propertyVariables := oldSchema.PropertyVariablesToReturn
	structuralVariables := oldSchema.StructuralVariablesToReturn
	usedNames := oldSchema.UsedNames
	hadToReturn := oldSchema.MustReturn

	*oldSchema = c.oldSchema

	oldSchema.UsedNames = usedNames

	if hadToReturn {
		for _, variable := range propertyVariables {
			oldSchema.AddPropertyVariable(variable)
		}
		for _, variable := range structuralVariables {
			oldSchema.AddStructuralVariable(variable)
		}
	}
}

type CallSubqueryWith struct {
	IsIncludeAll       bool
	VariablesToInclude []string
}

// Generate subclauses for CallSubqueryWith
func (c *CallSubqueryWith) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return nil
}

// TemplateString for CallSubqueryWith
func (c CallSubqueryWith) TemplateString() string {
	if c.IsIncludeAll {
		return "WITH *"
	}
	variablesAmount := len(c.VariablesToInclude)
	if variablesAmount == 0 {
		return ""
	}
	generatedString := "WITH " + c.VariablesToInclude[0]
	for i := 1; i < variablesAmount; i++ {
		generatedString += ", " + c.VariablesToInclude[i]
	}
	return generatedString
}
