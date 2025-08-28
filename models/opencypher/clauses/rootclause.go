package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

// A ReadClause is always succeeded another root clause, the only exception being [WriteClause].
type ReadClause struct {
	// If this is set, a generated WITH cannot generate additonal ORDER BY, SKIP or LIMIT clauses.
	// Used by CALL {} subquery and UNION
	SimpleWithClause bool
}

// Generate subclauses for RootClause
func (c *ReadClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	// Reset UseNewLabelMatch field
	s.UseNewLabelMatchType = nil

	if s.DisallowWriteClauses && seed.BooleanWithProbability(0.2) {
		return []translator.Clause{&Return{}}
	}

	// Choose clauses returning two root clauses with a lower probability, avoids infinite recursion
	if seed.BooleanWithProbability(0.15) {
		maxRes := 3
		if s.DisallowWriteClauses {
			maxRes = 2
		}
		switch seed.GetRandomIntn(maxRes) {
		case 0:
			return []translator.Clause{&CallSubquery{}}
		case 1:
			return []translator.Clause{&Union{}}
		case 2:
			return []translator.Clause{&Foreach{}}
		}
	}

	maxRes := 4
	if s.DisallowWriteClauses {
		maxRes = 3
	}
	switch seed.GetRandomIntn(maxRes) {
	case 0:
		return []translator.Clause{&Unwind{}}
	case 1:
		return []translator.Clause{&Match{}}
	case 2:
		return []translator.Clause{&With{SimpleWithClause: c.SimpleWithClause}}
	case 3:
		return []translator.Clause{&WriteClause{}}
	}
	return nil
}

// A WriteClause is always succeeded by an (optional) write clause, the only exception being [Return].
type WriteClause struct{}

// Generate subclauses for WriteQuery
func (c *WriteClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	// Reset UseNewLabelMatch field
	s.UseNewLabelMatchType = nil

	switch seed.GetRandomIntn(6) {
	case 0:
		return []translator.Clause{&Return{}}
	case 1:
		return []translator.Clause{&Create{}}
	case 2:
		return []translator.Clause{&Delete{}}
	case 3:
		return []translator.Clause{&Set{}}
	case 4:
		return []translator.Clause{&Merge{}}
	case 5:
		return []translator.Clause{&Remove{}}
	}
	return nil
}

type OptionalWriteQuery struct{}

// Generate subclauses for OptionalWriteQuery
func (c *OptionalWriteQuery) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if seed.RandomBoolean() {
		return []translator.Clause{&WriteClause{}}
	}
	if s.MustReturn {
		return []translator.Clause{&Return{}}
	}
	return nil
}

// The EmptyClause has no template string and no subclauses,
// thereby resulting in an empty string once translated.
type EmptyClause struct{}

// Generate subclauses for EmptyClause
func (c *EmptyClause) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	return nil
}
