package clauses

import (
	"fmt"

	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Transform a MatchRelationship to an equivalent using the fact that
//
//	()-[*x..y]-() => ()-[*x-z..y-z]-()-[*z..z]-()
func (c *MatchRelationship) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	// Split variable length matches:
	//  ()-[*x..y]-() => ()-[*x-z..y-z]-()-[*z..z]-()
	if (c.minVariableLength != nil || c.maxVariableLength != nil) && !c.hasStructureName {
		minLen := -1
		if c.minVariableLength != nil {
			minLen = *c.minVariableLength
		}
		if c.maxVariableLength != nil && (*c.maxVariableLength < minLen || minLen == -1) {
			minLen = *c.maxVariableLength
		}
		offset := 0
		if minLen >= 2 {
			offset = seed.GetRandomIntn(minLen-1) + 1
		} else {
			return nil
		}
		firstVariableLength := "*"
		secondVariableLength := "*"
		if c.minVariableLength != nil {
			firstVariableLength += fmt.Sprintf("%d", *c.minVariableLength-offset)
			secondVariableLength += fmt.Sprintf("%d", offset)
		}
		firstVariableLength += ".."
		secondVariableLength += ".."
		if c.maxVariableLength != nil {
			firstVariableLength += fmt.Sprintf("%d", *c.maxVariableLength-offset)
			secondVariableLength += fmt.Sprintf("%d", offset)
		}
		return helperclauses.CreateAssembler(
			fmt.Sprintf("[%%s%%s%s%%s]-()-[%%[1]s%%[2]s%s%%[3]s]", firstVariableLength, secondVariableLength),
			subclauses...,
		)
	}
	return nil
}
