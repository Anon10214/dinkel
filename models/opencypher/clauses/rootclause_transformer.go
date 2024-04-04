package clauses

import (
	"fmt"
	"strings"

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
		return helperclauses.CreateAssembler(
			subclauses,
			"WITH * %s",
		)
	}
	iteratorName := generateUniqueName(seed, s)
	if seed.RandomBoolean() {
		// UNWIND <x> AS I, if size(x) == 1
		expr := generateExpressionConf(seed)
		expr.IsList = true
		return helperclauses.CreateAssembler(
			[]translator.Clause{&Expression{Conf: expr}, &Expression{}, subclauses[0]},
			fmt.Sprintf("UNWIND (CASE size(%%s) WHEN 1 THEN %%[1]s ELSE [%%s] END) AS %s %%s", iteratorName),
		)
	}
	// <q><p> => <q> UNWIND x AS I <p>, if x not null and not a list
	expr := generateExpressionConf(seed)
	expr.TargetType = schema.PropertyValue
	expr.MustBeNonNull = true
	expr.IsList = false
	return helperclauses.CreateAssembler(
		[]translator.Clause{&Expression{Conf: expr}, subclauses[0]},
		fmt.Sprintf("UNWIND %%s AS %s %%s", iteratorName),
	)
}

// Transform a WriteClause to an equivalent using one of the following facts:
//   - FOREACH (I IN <null|[]> | ...) should never execute, including it should have no effect.
//   - Enclosing a WriteClause in a FOREACH, iterating over a single value shouldn't change the query's behavior.
//   - Enclosing a WriteClause in a WITH * should have no influence on its behavior, as variable aren't redefined.
//   - CREATE <x> DELETE <x> shouldn't change the query's behavior, as everything created in the first clause gets deleted right afterwards.
func (c WriteClause) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	// FOREACH (I IN <null|[]> | ...)
	if seed.BooleanWithProbability(0.33) {
		iteratorName := generateUniqueName(seed, s)
		if seed.RandomBoolean() {
			return helperclauses.CreateAssembler(
				[]translator.Clause{&ForeachCommand{}, subclauses[0]},
				fmt.Sprintf("FOREACH (%s IN %s | %%s) %%s", iteratorName, seed.RandomStringFromChoice("null", "[]")),
			)
		}
		Conf := generateExpressionConf(seed)
		Conf.IsList = true
		return helperclauses.CreateAssembler(
			[]translator.Clause{&Expression{Conf: Conf}, &ForeachCommand{}, subclauses[0]},
			fmt.Sprintf("FOREACH (%s IN (CASE size(%%s) WHEN 0 THEN %%[1]s ELSE %s END) | %%s) %%s", iteratorName, seed.RandomStringFromChoice("null", "[]")),
		)
	} else if seed.RandomBoolean() {
		return helperclauses.CreateAssembler(
			subclauses,
			"WITH * %s",
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
		append([]translator.Clause{path}, subclauses...),
		"CREATE %s "+deleteClause+" %s",
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
			[]translator.Clause{&Labels{LabelType: schema.NODE}, &Properties{}},
			"("+name+"%s%s)",
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
		[]translator.Clause{createNode, &Label{LabelType: schema.RELATIONSHIP}, &Properties{}, path},
		templateString,
	), vars
}
