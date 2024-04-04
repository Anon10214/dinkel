package clauses

import (
	"fmt"

	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Transform a PropertyLiteral to an equivalent through various means, depending on its type.
func (c *PropertyLiteral) Transform(seed *seed.Seed, s *schema.Schema, subclauses []translator.Clause) translator.Clause {
	// NULLs and aggregating functions mess everything up
	c.Conf.MustBeNonNull = true
	c.Conf.CanContainAggregatingFunctions = false
	switch c.value {
	case "true":
		if seed.RandomBoolean() {
			// x = x, with x not null
			c.Conf.MustBeNonNull = true
			return helperclauses.CreateAssembler(
				[]translator.Clause{&Expression{Conf: c.Conf}},
				"((%s)=(%[1]s))",
			)
		}
		// x OR true
		c.Conf.MustBeNonNull = false // null OR true = true
		return helperclauses.CreateAssembler(
			[]translator.Clause{&Expression{Conf: c.Conf}},
			seed.RandomStringFromChoice("(%s OR true)", "(true OR %s)"),
		)
	case "false":
		c.Conf.MustBeNonNull = false // null AND false = false
		return helperclauses.CreateAssembler(
			[]translator.Clause{&Expression{Conf: c.Conf}},
			seed.RandomStringFromChoice("(%s AND false)", "(false AND %s)"),
		)
	case "0": // No float 0.0, as multiplied by negative, gives -0.0 != 0.0
		return helperclauses.CreateAssembler(
			[]translator.Clause{&Expression{Conf: c.Conf}},
			fmt.Sprintf(seed.RandomStringFromChoice("(%%s * %s)", "(%s * %%s)"), c.value),
		)
	case "null":
		switch c.Conf.PropertyType {
		case schema.Boolean:
			if seed.RandomBoolean() {
				return helperclauses.CreateAssembler(
					[]translator.Clause{&Expression{Conf: c.Conf}},
					seed.RandomStringFromChoice("(null XOR %s)", "(%s XOR null)"),
				)
			}
			return helperclauses.CreateAssembler(
				[]translator.Clause{&Expression{}},
				seed.RandomStringFromChoice(
					"(null = %s)", "(%s = null)",
					"(null <> %s)", "(%s <> null)",
					"(null < %s)", "(%s < null)",
					"(null > %s)", "(%s > null)",
					"(null >= %s)", "(%s >= null)",
					"(null <= %s)", "(%s <= null)",
				))
		case schema.Float:
			if seed.BooleanWithProbability(0.25) {
				return helperclauses.CreateAssembler(
					[]translator.Clause{&Expression{Conf: c.Conf}},
					seed.RandomStringFromChoice("(%s^null)", "(null^%s)"),
				)
			}
			fallthrough
		case schema.Integer:
			if seed.BooleanWithProbability(0.2) {
				return helperclauses.CreateStringer("(-null)")
			}
			return helperclauses.CreateAssembler(
				[]translator.Clause{&Expression{Conf: c.Conf}},
				seed.RandomStringFromChoice(
					"(%s+null)", "(null+%s)",
					"(%s-null)", "(null-%s)",
					"(%s*null)", "(null*%s)",
					"(%s/null)", "(null/%s)",
				),
			)
		}
	}
	return nil
}
