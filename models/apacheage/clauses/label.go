package clauses

import (
	"github.com/Anon10214/dinkel/models/opencypher/clauses"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
)

// ExistingLabel drop-in for apache age
type ExistingLabel struct {
	LabelType schema.StructuralType
	name      string
}

// Generate the ExistingLabel subclauses.
// Different from the OpenCypher implementation, this drop-in never randomly chooses
// a label for a different structuralType than specified in its LabelType.
func (c *ExistingLabel) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	if len(s.Labels[c.LabelType]) > 0 {
		c.name = s.Labels[c.LabelType][seed.GetRandomIntn(len(s.Labels[c.LabelType]))]
		return nil
	}
	return []translator.Clause{&clauses.NewLabel{LabelType: c.LabelType}}
}

// TemplateString returns either the chosen name or a format string for a new label
func (c ExistingLabel) TemplateString() string {
	if c.name != "" {
		return c.name
	}
	return "%s"
}
