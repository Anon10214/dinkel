package helperclauses_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/Anon10214/dinkel/models/mock"
	"github.com/Anon10214/dinkel/models/opencypher"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
	"github.com/stretchr/testify/assert"
)

func generateMockClause(clause translator.Clause) string {
	return generateMockClauseWithSeed(clause, seed.GetRandomByteString())
}

func generateMockClauseWithSeed(clause translator.Clause, seed *seed.Seed) string {
	schema := &schema.Schema{}
	schema.Reset()

	res, _ := translator.GenerateStatement(
		seed,
		schema,
		clause,
		mock.Implementation{},
		0,
	)

	return res
}

func ExampleEmptyClause() {
	fmt.Println(
		len(generateMockClause(&helperclauses.EmptyClause{})),
	)
	// Output: 0
}

func ExampleStringer() {
	fmt.Println(
		generateMockClause(helperclauses.CreateStringer("ABC")),
	)
	// Output: ABC
}
func ExampleAssembler() {
	stringer1 := helperclauses.CreateStringer("hello")
	stringer2 := helperclauses.CreateStringer("world")

	subclauses := []translator.Clause{stringer1, stringer2}
	templateString := "%s - %s"

	fmt.Println(
		generateMockClause(
			helperclauses.CreateAssembler(templateString, subclauses...),
		),
	)
	// Output: hello - world
}
func ExampleAssembler_withoutTemplateString() {
	stringer1 := helperclauses.CreateStringer("no")
	stringer2 := helperclauses.CreateStringer("spaces")

	subclauses := []translator.Clause{stringer1, stringer2}

	fmt.Println(
		generateMockClause(
			helperclauses.CreateAssemblerWithoutTemplateString(subclauses...),
		),
	)
	// Output: nospaces
}

func ExampleClauseCapturer_regenerate() {
	// Get a clause capturer for an OpenCypher root clause
	helperclauses.SetImplementation(mock.Implementation{})
	clause := helperclauses.GetClauseCapturerForClause(&opencypher.RootClause{})

	// Generate a clause with a set underlying seed
	origSeed := seed.GetRandomByteStringWithSource(
		*rand.New(rand.NewSource(42)),
	)
	orig := generateMockClauseWithSeed(clause, origSeed)

	// Regenerate with a different underlying seed
	regeneratedSeed := seed.GetRandomByteStringWithSource(
		*rand.New(rand.NewSource(1337)),
	)
	regenerated := generateMockClauseWithSeed(clause, regeneratedSeed)

	// Regenerating the clause capturer results in the same clause
	fmt.Println(orig == regenerated)
	// Output: true
}

func TestStringer_EscapePercentage(t *testing.T) {
	assert.Equal(t, "%", generateMockClause(helperclauses.CreateStringer("%")), "Percentage signs were escaped incorrectly")
}

func TestHookClause_Empty(t *testing.T) {
	assert.Equal(t, generateMockClause(helperclauses.HookClause{}), "", "Default hook clause results in non-empty statement")
}

func TestHookClause_Populated(t *testing.T) {
	generateChan := make(chan struct{}, 1)
	templateStringChan := make(chan struct{}, 1)
	modifySchemaChan := make(chan struct{}, 1)

	generateHook := func(*seed.Seed, *schema.Schema) []translator.Clause { generateChan <- struct{}{}; return nil }
	templateStringHook := func() string { templateStringChan <- struct{}{}; return "" }
	modifySchemaHook := func(*schema.Schema) { modifySchemaChan <- struct{}{} }

	generateMockClause(helperclauses.HookClause{
		GenerateHook:       generateHook,
		TemplateStringHook: templateStringHook,
		ModifySchemaHook:   modifySchemaHook,
	})

	// Times out if the hooks aren't called in the right order
	<-generateChan
	<-templateStringChan
	<-modifySchemaChan
}
