package clauses

import (
	"fmt"
	"strings"

	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Transform a RootClause to an equivalent using one of the following facts:
//   - Enclosing a RootClause in a WITH * should have no influence on its behavior, as variable aren't redefined.
//   - Enclosing a RootClause in an UNWIND, iterating over a single value shouldn't change the query's behavior.
func (c ReadClause) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	if seed.RandomBoolean() {
		if config.GetConfig().AsteriskNeedsTargets {
			// Make sure we can actually generate a WITH * here
			if len(s.PropertyVariablesByName)+len(s.StructuralVariablesByName) == 0 {
				return nil
			}
		}
		return helperclauses.CreateAssembler(
			"WITH * %s",
			subclauses...,
		)
	}
	iteratorName := generateUniqueName(seed, s)
	if seed.RandomBoolean() {
		// UNWIND <x> AS I, if size(x) == 1
		expr := generateExpressionConf(seed)
		expr.IsList = true
		return helperclauses.CreateAssembler(
			fmt.Sprintf("UNWIND (CASE size(%%s) WHEN 1 THEN %%[1]s ELSE [%%s] END) AS %s %%s", iteratorName),
			&Expression{Conf: expr}, &Expression{}, subclauses[0],
		)
	}
	if seed.RandomBoolean() {
		return helperclauses.CreateAssembler(
			fmt.Sprintf("CALL { WITH * RETURN %%s AS %s } %%s", iteratorName),
			&Expression{}, subclauses[0],
		)
	}

	// This could explode the AST, keep probability low
	if !s.DisallowWriteClauses && seed.BooleanWithProbability(0.25) {
		// CALL { WITH * <DEAD CODE> }
		// If write clauses are disallowed, generation will never terminate, since the CALL subclause
		// cannot return, as this would otherwise influence the amount of rows returned by the query,
		// thus not being an equivalence transform.
		s.IsInSubquery = true
		return helperclauses.CreateAssembler(
			"CALL { WITH * %s } %s",
			&DeadCode{}, subclauses[0],
		)
	}

	if seed.RandomBoolean() {
		return helperclauses.CreateAssembler(
			"OPTIONAL MATCH %s %s",
			&NonexistantPattern{}, subclauses[0],
		)
	}

	// <q><p> => <q> UNWIND x AS I <p>, if x not null and not a list
	expr := generateExpressionConf(seed)
	expr.TargetType = schema.PropertyValue
	expr.MustBeNonNull = true
	expr.IsList = false
	return helperclauses.CreateAssembler(
		fmt.Sprintf("UNWIND %%s AS %s %%s", iteratorName),
		&Expression{Conf: expr}, subclauses[0],
	)
}

// Transform a WriteClause to an equivalent using one of the following facts:
//   - FOREACH (I IN <null|[]> | ...) should never execute, including it should have no effect.
//   - Enclosing a WriteClause in a FOREACH, iterating over a single value shouldn't change the query's behavior.
//   - Enclosing a WriteClause in a WITH * should have no influence on its behavior, as variable aren't redefined.
//   - DELETE <x> on an already deleted x shouldn't have an effect
//   - CREATE <x> DELETE <x> shouldn't change the query's behavior, as everything created in the first clause gets deleted right afterwards.
func (c WriteClause) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	switch seed.GetRandomIntn(3) {
	// FOREACH (I IN <null|[]> | ...)
	case 0:
		iteratorName := generateUniqueName(seed, s)
		if seed.RandomBoolean() {
			return helperclauses.CreateAssembler(
				fmt.Sprintf("FOREACH (%s IN %s | %%s) %%s", iteratorName, seed.RandomStringFromChoice("null", "[]")),
				&ForeachCommand{}, subclauses[0],
			)
		}
		Conf := generateExpressionConf(seed)
		Conf.IsList = true
		return helperclauses.CreateAssembler(
			fmt.Sprintf("FOREACH (%s IN (CASE size(%%s) WHEN 0 THEN %%[1]s ELSE %s END) | %%s) %%s", iteratorName, seed.RandomStringFromChoice("null", "[]")),
			&Expression{Conf: Conf}, &ForeachCommand{}, subclauses[0],
		)
		// WITH *
	case 1:
		if config.GetConfig().AsteriskNeedsTargets {
			// Make sure we can actually generate a WITH * here
			if len(s.PropertyVariablesByName)+len(s.StructuralVariablesByName) == 0 {
				return nil
			}
		}
		return helperclauses.CreateAssembler(
			"WITH * %s",
			subclauses...,
		)
		// DELETE <x>
	case 2:
		var toDelete []string
		for seed.RandomBoolean() {
			for delVar := range s.DeletedVars {
				if seed.RandomBoolean() {
					toDelete = append(toDelete, delVar)
				}
			}
		}

		if len(toDelete) == 0 {
			return nil
		}

		clause := seed.RandomStringFromChoice("DELETE ", "DETACH DELETE ") + toDelete[0]
		for _, delVar := range toDelete[1:] {
			clause += ", " + delVar
		}

		return helperclauses.CreateAssembler(
			clause+" %s",
			subclauses...,
		)
	}
	//	CREATE <x> DELETE <x>
	//
	// Should have no outcome on the result, if the CREATE clause only creates new elements
	path, vars := createNewPathWithVariables(seed, s, nil)
	var deleteClause string
	if seed.RandomBoolean() {
		deleteClause += "DETACH "
	}
	var varsAsAny []any
	for _, variable := range vars {
		varsAsAny = append(varsAsAny, variable)
	}
	deleteClause += fmt.Sprintf("DELETE %s"+strings.Repeat(", %s", len(vars)-1), varsAsAny...)
	return helperclauses.CreateAssembler(
		"CREATE %s "+deleteClause+" %s",
		append([]translator.Clause{path}, subclauses...)...,
	)
}

// Returns a path expression where every element is new and has a variable.
// Takes in a list of variables that got created in the same clause and can be referenced.
// Returned slice is guaranteed to be non-empty (need to create at least one element).
func createNewPathWithVariables(seed *seed.Seed, s *schema.Schema, usedVars []string) (translator.Clause, []string) {
	var createNode translator.Clause
	if len(usedVars) != 0 && seed.RandomBoolean() {
		// Reference a previously created node
		createNode = helperclauses.CreateStringer("(" + seed.RandomStringFromChoice(usedVars...) + ")")
	} else {
		// Create a new node
		name := generateUniqueName(seed, s)
		usedVars = append(usedVars, name)
		createNode = helperclauses.CreateAssembler(
			"("+name+"%s%s)",
			&Labels{LabelType: schema.NODE}, &Properties{},
		)
	}

	if seed.RandomBoolean() {
		// Stop creation of more elements
		return createNode, usedVars
	}
	// Create relationship and additional nodes
	relationshipName := generateUniqueName(seed, s)
	path, vars := createNewPathWithVariables(seed, s, usedVars)

	// Add relationship name after recursion so they don't get mistaken for node names
	vars = append(vars, relationshipName)

	templateString := "%s"

	if seed.RandomBoolean() {
		templateString += "<-[" + relationshipName + "%s%s]-"
	} else {
		templateString += "-[" + relationshipName + "%s%s]->"
	}
	templateString += "%s"

	return helperclauses.CreateAssembler(
		templateString,
		createNode, &Label{LabelType: schema.RELATIONSHIP}, &Properties{}, path,
	), vars
}
