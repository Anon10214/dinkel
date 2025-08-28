package helperclauses

import (
	"reflect"
	"strings"

	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/qdm12/reprint"
	"github.com/sirupsen/logrus"
)

var implementation translator.Implementation

// SetImplementation takes in a [translator.Implementation] and stores it for subsequent [ClauseCapturer]-s to use.
func SetImplementation(impl translator.Implementation) {
	implementation = impl
}

// The ClauseCapturer essentially represents a clause singleton.
//
// It takes in a clause and generates it when generating itself.
// From then on, every time the capturer get regenerated, it results in the exact
// same clause again, causing no changes in the schema, seed or its translated result.
//
// Any side effects produced during the captured clause's generation won't be reapplied
// on subsequent invocations of the generation function.
//
// Changes directly affecting the passed clause that got captured will be reflected in
// all other capturers with this captured clause.
//
// This is useful if you want to have different subsequent or preceding clauses,
// but want to keep one clause the same.
type ClauseCapturer struct {
	// If this clause was already generated
	generated bool
	// If this clause is being regenerated
	regenerated bool
	// The schema at the time of generation, to be used when regenerating
	capturedSchema *schema.Schema
	// The schema after the modifications, current schema will be set to this in the modifySchema call
	capturedModifiedSchema *schema.Schema
	// The captured clause that gets generated
	capturedClause translator.Clause
	// The (captured) subclauses returned by the captured clause
	subclauses []*ClauseCapturer
	// The drop ins of the current implementation
	dropIns translator.DropIns
}

// GetClauseCapturerForClause returns a new capturer for a clause and an implementation.
// Before calling thus function, [SetImplementation] has to have been called for initializing the used implementation.
// Otherwise, this function will panic.
func GetClauseCapturerForClause(clause translator.Clause) *ClauseCapturer {
	if implementation == nil {
		logrus.Panicf("Clause capturer implementation is nil, but a clause capturer is being created. Did you call helperclauses.SetImplementation?")
	}
	return getClauseCapturerForClause(clause, implementation.GetDropIns())
}

func getClauseCapturerForClause(clause translator.Clause, dropIns translator.DropIns) *ClauseCapturer {
	// Return the clause itself if it is already a clause capturer
	if capturer, ok := clause.(*ClauseCapturer); ok {
		return capturer
	}

	return &ClauseCapturer{
		capturedClause: clause,
		dropIns:        dropIns,
	}
}

// GetCapturedClause returns the underlying captured clause
func (c *ClauseCapturer) GetCapturedClause() translator.Clause {
	return c.capturedClause
}

// GetSubclauseClauseCapturers returns the subclauses of the clause capturer, which are
// themselves also clause capturers. This will return nil if the clause hasn't been generated yet.
func (c ClauseCapturer) GetSubclauseClauseCapturers() []*ClauseCapturer {
	return c.subclauses
}

// GetCapturedSchema returns the schema the clause was originally generated with.
// Guaranteed to be non nil if the clause was once generated and not updated later.
func (c ClauseCapturer) GetCapturedSchema() *schema.Schema {
	return c.capturedSchema
}

// UpdateClause sets the captured clause to the passed clause.
//
// Calling this method also sets generated to false, so if this ClauseCapturer's generate
// function gets invoked again, the captured clause gets regenerated again.
//
// When regenerating, the schema which generated the original clause gets reused.
// If this is undesired, use RenewClause instead.
func (c *ClauseCapturer) UpdateClause(newClause translator.Clause) {
	if capturer, ok := newClause.(*ClauseCapturer); ok {
		*c = *capturer.Copy()
		return
	}
	c.generated = false
	c.subclauses = nil
	c.regenerated = true
	c.capturedClause = newClause
}

// RenewClause sets the captured clause to the passed clause.
//
// Calling this method also sets generated to false, so if this ClauseCapturer's generate
// function gets invoked again, the captured clause gets regenerated again.
//
// When regenerating, the schema passed during generation of the new clause gets used instead of the one
// used when originally generating the clause. If this is undesired, use UpdateClause instead.
func (c *ClauseCapturer) RenewClause(newClause translator.Clause) {
	c.generated = false
	c.subclauses = nil
	c.regenerated = false
	c.capturedSchema = nil
	c.capturedModifiedSchema = nil
	c.capturedClause = newClause
}

// Copy returns a deep copy of the clause capturer.
// Descendants of the returned clause are also deep copied and fully uncoupled from the descendants of the passed clause capturer.
func (c ClauseCapturer) Copy() *ClauseCapturer {
	var subclauses []*ClauseCapturer
	for _, subclause := range c.subclauses {
		subclauses = append(subclauses, subclause.Copy())
	}
	return &ClauseCapturer{
		generated:              c.generated,
		regenerated:            c.regenerated,
		capturedSchema:         c.capturedSchema.Copy(),
		capturedModifiedSchema: c.capturedModifiedSchema.Copy(),
		capturedClause:         reprint.This(c.capturedClause).(translator.Clause),
		subclauses:             subclauses,
	}
}

// GenerateAST pre-generates the whole AST of the clause capturer,
// thus uncoupling its generation from the translator.
func (c *ClauseCapturer) GenerateAST(seed *seed.Seed, s *schema.Schema) {
	c.Generate(seed, s)
	for _, subclause := range c.subclauses {
		subclause.GenerateAST(seed, s)
	}
	c.ModifySchema(s)
}

// Generate the ClauseCapturer's subclauses.
//
// The first call to this method generates the captured clause and return its subclauses.
// Subsequent invocations, as long as the captured clause wasn't changed, cause these same
// subclauses to be returned.
func (c *ClauseCapturer) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	logrus.Tracef("Clause capturer generating %T, %#v", c.capturedClause, c.capturedClause)
	if !c.generated {
		c.adaptClauseToImplementation(seed, s)
		// Generate the subclauses
		var subclauses []translator.Clause
		if c.regenerated {
			// Set the current schema to the one used originally when generating
			*s = *c.capturedSchema
			// Reuse the captured schema if regenerated
			subclauses = c.capturedClause.Generate(seed, s)
		} else {
			c.capturedSchema = s.Copy()
			subclauses = c.capturedClause.Generate(seed, s)
		}

		for _, subclause := range subclauses {
			c.subclauses = append(c.subclauses, getClauseCapturerForClause(subclause, c.dropIns))
		}
	}
	var subclausesAsClauses []translator.Clause
	for _, subclause := range c.subclauses {
		subclausesAsClauses = append(subclausesAsClauses, subclause)
	}
	return subclausesAsClauses
}

// TemplateString returns the captured clause's template string, if defined.
// Else, TemplateString returns the default template string, causing
// the subclauses to simply get concatenated.
func (c ClauseCapturer) TemplateString() string {
	if templater, ok := c.capturedClause.(translator.Templater); ok {
		return templater.TemplateString()
	}
	return strings.Repeat("%s", len(c.subclauses))
}

// ModifySchema invokes the captured clause's ModifySchema functions, if defined and if the clause
// hasn't been generated before.
//
// It then sets generated to true, to ensure that the generation of the clause capturer returns
// the same results from now on.
func (c *ClauseCapturer) ModifySchema(s *schema.Schema) {
	if !c.generated {
		if modifier, ok := c.capturedClause.(translator.PostGenerationSchemaModifier); ok {
			modifier.ModifySchema(s)
		}
		c.capturedModifiedSchema = s.Copy()
	} else {
		*s = *c.capturedModifiedSchema.Copy()
	}
	c.generated = true
}

func (c *ClauseCapturer) adaptClauseToImplementation(seed *seed.Seed, s *schema.Schema) {
	clause := c.capturedClause

	if fun, ok := c.dropIns[reflect.TypeOf(c.capturedClause)]; ok {
		newClause := fun(c.capturedClause, seed, s)
		logrus.Tracef("Adapting captured clause of type %T to type %T", c.capturedClause, newClause)
		clause = newClause
	}

	c.capturedClause = clause
}
