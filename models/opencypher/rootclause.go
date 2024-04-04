/*
Package opencypher provides the OpenCypher model from which all other models descend.

This model is the only model without an implementation or driver.
All other models essentially extend the OpenCypher model by modifying
its [clauses], generation [config.Config] and [schema.OpenCypherSchema].
*/
package opencypher

import (
	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"

	// Imports for docstring
	"fmt"

	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
)

// RootClause of OpenCypher represents the root of the AST.
type RootClause clauses.ReadClause

// Generate subclauses for OpenCypherRootClause
func (c *RootClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if out := seed.GetByte(); out%5 == 0 {
		return []translator.Clause{&clauses.Index{}}
	}
	return []translator.Clause{&clauses.ReadClause{}}
}

// Function s.t. docstring points to the right code
func _() {
	fmt.Print(config.Config{}, schema.Schema{})
}
