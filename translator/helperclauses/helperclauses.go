/*
Package helperclauses provides clauses that simplify the modelling of new queries.

These clauses are not implementation specific and don't modify the schema
or seed on their own, unless subqueries with such behavior are supplied.
*/
package helperclauses

import (
	"strings"

	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

// The EmptyClause simply results in an empty string during generation.
// It returns no subclauses and doesn't modify the schema or seed.
type EmptyClause struct{}

// Generate the EmptyClause subclauses, does nothing and returns nil
func (c *EmptyClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return nil
}

// A Stringer has no subclauses and does not modify the schema or seed.
// It simply returns the saved string as its template string.
type Stringer struct {
	value string
}

// CreateStringer returns a stringer generating the passed string.
func CreateStringer(val string) *Stringer {
	val = strings.ReplaceAll(val, "%", "%%")
	return &Stringer{value: val}
}

// Generate the Stringer subclauses, does nothing and returns nil
func (c Stringer) Generate(*seed.Seed, *schema.Schema) []translator.Clause {
	return nil
}

// TemplateString returns the Stringer's template string, which is just its associated value
func (c Stringer) TemplateString() string {
	return c.value
}

// An Assembler returns the provided subclauses during generation and the provided template string when prompted.
// The assembler itself does not modify the schema or seed, though the subclauses might.
type Assembler struct {
	subclauses     []translator.Clause
	templateString string
}

// CreateAssembler returns an assembler given the passed subclauses and template string
func CreateAssembler(templateString string, subclauses ...translator.Clause) *Assembler {
	return &Assembler{
		subclauses:     subclauses,
		templateString: templateString,
	}
}

// CreateAssemblerWithoutTemplateString returns an assembler whose generated result will consist of the subclauses concatenated.
func CreateAssemblerWithoutTemplateString(subclauses ...translator.Clause) *Assembler {
	return &Assembler{
		subclauses:     subclauses,
		templateString: strings.Repeat("%s", len(subclauses)),
	}
}

// Generate the assembler's subclauses, returns the subclauses the assembler was initialized with.
func (c Assembler) Generate(*seed.Seed, *schema.Schema) []translator.Clause {
	return c.subclauses
}

// TemplateString returns the template string the assembler was initialized with.
func (c Assembler) TemplateString() string {
	return c.templateString
}

// A HookClause calls provided hooks. It doesn't provide any value for fuzzing and is supposed to be used for testing.
type HookClause struct {
	GenerateHook       func(*seed.Seed, *schema.Schema) []translator.Clause
	TemplateStringHook func() string
	ModifySchemaHook   func(*schema.Schema)
}

// Generate returns nil and calls the GenerateHook if defined
func (c HookClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if c.GenerateHook != nil {
		return c.GenerateHook(seed, s)
	}
	return nil
}

// TemplateString returns the empty string and calls the TemplateStringHook if defined.
func (c HookClause) TemplateString() string {
	if c.TemplateStringHook != nil {
		return c.TemplateStringHook()
	}
	return ""
}

// ModifySchema does nothing and calls the ModifySchemaHook if defined
func (c HookClause) ModifySchema(s *schema.Schema) {
	if c.ModifySchemaHook != nil {
		c.ModifySchemaHook(s)
	}
}
