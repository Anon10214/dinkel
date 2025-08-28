/*
Package translator provides translations from ASTs to concrete queries.

The translator package holds everything related to AST nodes ([Clause])
and handles translation of the AST to a concrete query.
*/
package translator

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/sirupsen/logrus"
)

// A DropIn for an OpenCypher clause.
//
// It takes in an OpenCypher clause and returns
// a modified clause adapted to the specific implementation
type DropIn func(Clause, *seed.Seed, *schema.Schema) Clause

// DropIns map clause types to drop-ins used by implementations to fine tune generation.
type DropIns map[reflect.Type]DropIn

// An Implementation represents a concrete OpenCypher implementation.
type Implementation interface {
	// Returns the implementation's drop-ins
	GetDropIns() DropIns
	// Returns the language-specific config to be passed to OpenCypher
	GetOpenCypherConfig() config.Config
}

// A Clause from a query, makes up an element in the AST
type Clause interface {
	// Generate the clause's subclauses given a seed and the current state
	Generate(*seed.Seed, *schema.Schema) []Clause
}

// A Templater is a clause that combines its subclauses in a non-straightforward way.
type Templater interface {
	Clause
	// The template string as passed to fmt.Sprintf with the subclauses as arguments
	TemplateString() string
}

// A PostGenerationSchemaModifier is a clause that changes the schema in some way after its subclauses have generated
type PostGenerationSchemaModifier interface {
	Clause
	// Modifies the schema, gets called after subclauses have been generated
	ModifySchema(*schema.Schema)
}

// A Transformer provides a function to return a clause that is equivalent to itself.
// Used for logic bug fuzzing via equivalence transformation.
type Transformer interface {
	Clause
	// Gets called after generation and if the strategy is set to test for equivalence logic bugs.
	// Gets passed the seed, the schema when the original clause first got generated and the subclauses it generated.
	// If the functions returns nil, the clause won't be updated and the old one will be kept.
	Transform(*seed.Seed, *schema.Schema, []Clause) Clause
}

type ASTNodeLimitReached error

// GenerateStatement generates a statement given a seed, initial schema, the model's root clause and the specific OpenCypher implementation.
// An additional parameter defines how many ast nodes can be generated at most (or <= 0 if no limit), after which the generation terminates and an error is thrown.
// The only error that can be returned from this function is of type [ASTNodeLimitReached].
func GenerateStatement(seed *seed.Seed, schema *schema.Schema, rootClause Clause, implementation Implementation, maxASTNodes int64) (string, error) {
	// Set the generation config
	config.SetConfig(implementation.GetOpenCypherConfig())

	if maxASTNodes <= 0 {
		maxASTNodes = math.MaxInt64
	}

	stmnt, nodes := generateStatement(seed, schema, rootClause, implementation, maxASTNodes)

	if nodes >= maxASTNodes {
		return "", ASTNodeLimitReached(errors.New("ast node limit reached"))
	}
	return stmnt, nil
}

// Generate a statement given a seed, schema, the current clause and the specific OpenCypher implementation.
// This function returns the part of the statement that was generated as well as how many nodes the AST contains.
func generateStatement(seed *seed.Seed, schema *schema.Schema, clause Clause, implementation Implementation, maxASTNodes int64) (string, int64) {
	if maxASTNodes == 1 {
		// Terminate if limit reached
		return "", 1
	}

	clause = adaptClauseToImplementation(clause, implementation, seed, schema)
	subclauses := clause.Generate(seed, schema)

	logrus.Tracef("Generating %T", clause)

	// Get the template string, simply combining subclauses if none is given
	var templateString string
	switch c := clause.(type) {
	case Templater:
		templateString = c.TemplateString()
	default:
		templateString = strings.Repeat("%s", len(subclauses))
	}

	var nodes int64 = 1
	var subclausesAsStrings []any // Needs to be of type []any s.t. fmt.Sprintf can accept it
	for _, subclause := range subclauses {
		subclauseStr, nodesCnt := generateStatement(seed, schema, subclause, implementation, maxASTNodes-nodes)
		nodes += nodesCnt
		if nodes >= maxASTNodes {
			return "", nodes
		}
		subclausesAsStrings = append(subclausesAsStrings, subclauseStr)
	}

	// Let the clause modify the schema if it is a PostGenerationSchemaModifier
	asSchemaModifier, isSchemaModifier := clause.(PostGenerationSchemaModifier)
	if isSchemaModifier {
		asSchemaModifier.ModifySchema(schema)
	}

	logrus.Tracef("Done generating %T", clause)

	return fmt.Sprintf(templateString, subclausesAsStrings...), nodes
}

// Returns the clause replaced with the implementation-specific drop-in.
// If there is no drop-in for the passed clause, no drop-in is specified, the clause itself is returned.
func adaptClauseToImplementation(clause Clause, implementation Implementation, seed *seed.Seed, schema *schema.Schema) Clause {
	if fun, ok := implementation.GetDropIns()[reflect.TypeOf(clause)]; ok {
		return fun(clause, seed, schema)
	}
	return clause
}
