package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Transform a FunctionApplicationExpression to an equivalent by returning
// expressions evaluating to the same result as calling the function.
func (c FunctionApplicationExpression) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	if c.target == nil {
		return nil
	}
	switch c.target.Name {
	case "abs":
		// Break up into CASE statement
		return helperclauses.CreateAssembler(
			"(CASE WHEN (%s) < 0 THEN -(%[1]s) ELSE %[1]s END)",
			subclauses...,
		)
	case "toBoolean":
		// toBoolean(x:boolean) = x
		if c.target.InputTypes[0].PropertyType == schema.Boolean {
			return helperclauses.CreateAssemblerWithoutTemplateString(subclauses...)
		}
	case "toFloat":
		// toFloat(x:float) = x
		if c.target.InputTypes[0].PropertyType == schema.Float {
			return helperclauses.CreateAssemblerWithoutTemplateString(subclauses...)
		}
	case "toInteger":
		// toInteger(x:int) = x
		if c.target.InputTypes[0].PropertyType == schema.Integer {
			return helperclauses.CreateAssemblerWithoutTemplateString(subclauses...)
		}
	}
	return nil
}
