/*
A cypher fuzzer generating complex, semantically and syntactically valid queries.

Dinkel provides an easily expandable framework for targeting all possible cypher
implementations. Additionally, dinkel supports different fuzzing techniques,
targeting exception and logic bugs.

This allows for easy and thorough testing of any cypher implementation with
little setup required.

Dinkel achieves query complexity and validity by keeping track of the query
context and database state during generation. This information then gets used
within the stateful generation of a query's clauses, allowing for complex data
dependencies within a query.
*/
package main

import (
	_ "embed"

	"github.com/Anon10214/dinkel/cmd"
)

func main() {
	initEmbeds()
	cmd.Execute()
}

// Embed the targets-config.yml content such that the config command can reuse it
//
//go:embed targets-config.yml
var targetConfigTemplate string

func initEmbeds() {
	cmd.TargetConfigTemplate = targetConfigTemplate
}
