package translator_test

import (
	"math/rand"
	"testing"

	"github.com/Anon10214/dinkel/models/mock"
	"github.com/Anon10214/dinkel/models/opencypher"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

const (
	samples int = 10000
)

// Test generation of OpenCypherRootClauses with a mock implementation.
// This should achieve close to 100% code coverage in [models/opencypher/clauses]
// , responsible for clause generation. Ignoring the code responsible for adapting
// to an implementation. If this is not the case, then there is dead code present.
func TestGeneration(t *testing.T) {
	// Run a few generations to ensure no crashes or infinite loops occur
	for i := int64(0); i < int64(samples); i++ {
		seed := seed.GetRandomByteStringWithSource(*rand.New(rand.NewSource(i)))

		schema := &schema.Schema{}
		schema.Reset()

		translator.GenerateStatement(seed, schema, &opencypher.RootClause{}, mock.Implementation{}, 0)
	}
}
