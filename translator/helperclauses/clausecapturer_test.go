package helperclauses_test

import (
	"math/rand"
	"testing"

	"github.com/Anon10214/dinkel/models/mock"
	"github.com/Anon10214/dinkel/models/opencypher"
	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
	"github.com/stretchr/testify/assert"
)

// Ensure that clauseCapturer.Copy returns a deep copy
func TestClauseCapturer_Copy(t *testing.T) {
	// Get a clause capturer for an OpenCypher root clause
	helperclauses.SetImplementation(mock.Implementation{})
	clause := helperclauses.GetClauseCapturerForClause(&opencypher.RootClause{})

	seed := seed.GetRandomByteStringWithSource(
		*rand.New(rand.NewSource(123)),
	)
	generateMockClauseWithSeed(clause, seed)

	clauseCopy := clause.Copy()

	// Assert the clauses still hold the same values but are distinct
	assertClausesEqualButDistinct(t, clause, clauseCopy)
}

// Asserts that all of the clauses subclauses and recursively their subclauses are equal but
// point to dinstinct instances.
func assertClausesEqualButDistinct(t *testing.T, a *helperclauses.ClauseCapturer, b *helperclauses.ClauseCapturer) {
	// Clauses have the same values but are dinstinct instances
	assert.Equal(t, a, b, "Clauses do not have the same underlying values")
	assert.NotSame(t, a, b, "Clause capturers point to the same memory address")

	// Assert that the clause's subclauses are equal but distinct
	assert.Equal(t, len(a.GetSubclauseClauseCapturers()), len(b.GetSubclauseClauseCapturers()))
	for i := range a.GetSubclauseClauseCapturers() {
		assertClausesEqualButDistinct(t,
			a.GetSubclauseClauseCapturers()[i],
			b.GetSubclauseClauseCapturers()[i],
		)
	}
}

func TestClauseCapturer_SchemaPreserved(t *testing.T) {
	// Get a clause capturer for a state dependent clause
	helperclauses.SetImplementation(mock.Implementation{})
	clause := helperclauses.GetClauseCapturerForClause(&clauses.ExistingLabel{})

	// This assembler ensures that the schema gets modified before and after the clause does
	setupClauses := helperclauses.CreateAssemblerWithoutTemplateString(
		&clauses.NewLabel{},
		clause,
	)

	origSeed := seed.GetRandomByteStringWithSource(
		*rand.New(rand.NewSource(0xdeadbeef)),
	)
	origStatement := generateMockClauseWithSeed(setupClauses, origSeed)

	// Update the clause
	clause.UpdateClause(&clauses.ExistingLabel{})

	// Regenerate the clause with an emtpy schema
	// The seed is irrelevant as the clause can only choose from one label at most
	updatedStatement := generateMockClause(clause)
	// If the orig statement was "LABELLABEL", then the updated statement must be "LABEL"
	expected := updatedStatement + updatedStatement

	assert.Equal(t, origStatement, expected, "Schema was not preserved")
}

// Ensure that the schema is preserved when updating a clause.
//
// Test this by generating a NewLabel clause, thus populating the schema with
// a single possible label. Then, capture an EmptyClause. The capturer should
// store the schema arrived there.
//
// Update the captured clause to be an ExistingLabel clause and only
// generate the clause capturer. If the schema was preserved, the only existing
// label ExistingLabel can choose from is the one generated before.
// If the generated queries now match, then we can confirm that the schema was preserved.
func TestClauseCapturer_UpdateClauseSchemaPreserved(t *testing.T) {
	helperclauses.SetImplementation(mock.Implementation{})
	clause := helperclauses.GetClauseCapturerForClause(&clauses.EmptyClause{})

	// This assembler ensures that the schema gets modified before and after the clause does
	setupClauses := helperclauses.CreateAssemblerWithoutTemplateString(
		&clauses.NewLabel{},
		clause,
	)

	origSeed := seed.GetRandomByteStringWithSource(
		*rand.New(rand.NewSource(0xdeadbeef)),
	)
	origStatement := generateMockClauseWithSeed(setupClauses, origSeed)

	// Update the clause to an existing label
	clause.UpdateClause(&clauses.ExistingLabel{})

	// Regenerate the clause with an emtpy schema
	// The seed is irrelevant as the clause can only choose from one label at most
	updatedStatement := generateMockClause(clause)

	assert.Equal(t, origStatement, updatedStatement, "Schema was not preserved")
}

// Ensure that the schema is not preserved when renewing a captured clause
func TestClauseCapturer_RenewClauseSchemaNotPreserved(t *testing.T) {
	schemaChan := make(chan *schema.Schema, 1)

	schemaHook := func(_ *seed.Seed, s *schema.Schema) []translator.Clause {
		schemaChan <- s
		return nil
	}

	hookClause := helperclauses.HookClause{
		GenerateHook: schemaHook,
	}
	helperclauses.SetImplementation(mock.Implementation{})
	capturedHookClause := helperclauses.GetClauseCapturerForClause(hookClause)

	setupClauses := helperclauses.CreateAssemblerWithoutTemplateString(
		// Add a label to the schema
		&clauses.NewLabel{},
		capturedHookClause,
	)

	origSeed := seed.GetRandomByteStringWithSource(
		*rand.New(rand.NewSource(0xdeadbeef)),
	)
	generateMockClauseWithSeed(setupClauses, origSeed)

	origLabels := (<-schemaChan).Labels[schema.ANY]
	assert.Len(t, origLabels, 1, "Less or more than 1 label was generated")

	capturedHookClause.RenewClause(hookClause)
	generateMockClause(hookClause)

	newLabels := (<-schemaChan).Labels[schema.ANY]
	assert.Len(t, newLabels, 0, "Schema was not reset")
}

// Test to verify that a captured clause still gets transformed to the implementation specific clause
func TestClauseCapturer_RespectsImplementation(t *testing.T) {
	impl := mock.Implementation{}

	// EmptyClause -> Stringer("ABC")
	impl.AddDropIn(&clauses.EmptyClause{}, func(translator.Clause, *seed.Seed, *schema.Schema) translator.Clause {
		return helperclauses.CreateStringer("ABC")
	})

	helperclauses.SetImplementation(impl)
	clause := helperclauses.GetClauseCapturerForClause(&clauses.EmptyClause{})

	schema := &schema.Schema{}
	schema.Reset()

	seed := seed.GetRandomByteStringWithSource(*rand.New(rand.NewSource(0x123)))

	res, _ := translator.GenerateStatement(
		seed,
		schema,
		clause,
		impl,
		0,
	)

	assert.Equal(t, "ABC", res, "Clause capturer did not respect implementation drop in")
}

func TestCapturedClause(t *testing.T) {
	clause := &helperclauses.EmptyClause{}

	helperclauses.SetImplementation(mock.Implementation{})
	captured := helperclauses.GetClauseCapturerForClause(clause)

	assert.Same(t, clause, captured.GetCapturedClause(), "Just captured clause is not the same as the passed one")

	generateMockClause(captured)

	assert.Same(t, clause, captured.GetCapturedClause(), "Captured clause is not the same as the original one after generation")
}

// Getting a clause capturer for a clause capturer
// should just return itself again.
func TestClauseCapturerOfClauseCapturer(t *testing.T) {
	helperclauses.SetImplementation(mock.Implementation{})
	clause := helperclauses.GetClauseCapturerForClause(&clauses.EmptyClause{})

	captured := helperclauses.GetClauseCapturerForClause(clause)

	assert.Same(t, clause, captured, "Clause capturer for clause capturer didn't return itself")
}

// Captured schema must be equal but not the same
func TestCapturedSchema(t *testing.T) {
	schemaChan := make(chan *schema.Schema, 1)

	schemaHook := func(_ *seed.Seed, s *schema.Schema) []translator.Clause {
		schemaChan <- s
		return nil
	}

	hookClause := helperclauses.HookClause{
		GenerateHook: schemaHook,
	}
	helperclauses.SetImplementation(mock.Implementation{})
	capturedHookClause := helperclauses.GetClauseCapturerForClause(hookClause)

	// Generate a root clause to populate the schema, then check it with
	// the hook clause
	statement := helperclauses.CreateAssemblerWithoutTemplateString(
		&opencypher.RootClause{},
		capturedHookClause,
	)

	seed := seed.GetRandomByteStringWithSource(
		*rand.New(rand.NewSource(424242)),
	)
	generateMockClauseWithSeed(statement, seed)

	schemaAtGeneration := <-schemaChan
	capturedSchema := capturedHookClause.GetCapturedSchema()

	assert.Equal(t, schemaAtGeneration, capturedSchema, "Captured schema doesn't match schema at generation")
	assert.NotSame(t, schemaAtGeneration, capturedSchema, "Captured schema doesn't point to separate memory address")
}
