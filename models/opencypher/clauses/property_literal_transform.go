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
				"((%s)=(%[1]s))",
				&Expression{Conf: c.Conf},
			)
		}
		// x OR true
		c.Conf.MustBeNonNull = false // null OR true = true
		return helperclauses.CreateAssembler(
			seed.RandomStringFromChoice("(%s OR %s)", "(%[2]s OR %[1]s)"),
			&Expression{Conf: c.Conf}, &Tautum{conf: c.Conf},
		)
	case "false":
		c.Conf.MustBeNonNull = false // null AND false = false
		return helperclauses.CreateAssembler(
			seed.RandomStringFromChoice("(%s AND %s)", "(%[2]s AND %[1]s)"),
			&Expression{Conf: c.Conf}, &Falsum{conf: c.Conf},
		)
	case "0": // No float 0.0, as multiplied by negative, gives -0.0 != 0.0
		return helperclauses.CreateAssembler(
			fmt.Sprintf(seed.RandomStringFromChoice("(%%s * %s)", "(%s * %%s)"), c.value),
			&Expression{Conf: c.Conf},
		)
	case "null":
		switch c.Conf.PropertyType {
		case schema.Boolean:
			if seed.RandomBoolean() {
				if seed.RandomBoolean() {
					// NULL AND tautum, NULL OR falsum
					if seed.RandomBoolean() {
						return helperclauses.CreateAssembler(
							seed.RandomStringFromChoice("(null AND %s)", "(%s AND null)"),
							&Tautum{},
						)
					}
					return helperclauses.CreateAssembler(
						seed.RandomStringFromChoice("(null OR %s)", "(%s OR null)"),
						&Falsum{},
					)
				}
				// NULL XOR b
				return helperclauses.CreateAssembler(
					seed.RandomStringFromChoice("(null XOR %s)", "(%s XOR null)"),
					&Expression{Conf: c.Conf},
				)
			}
			return helperclauses.CreateAssembler(
				seed.RandomStringFromChoice(
					"(null = %s)", "(%s = null)",
					"(null <> %s)", "(%s <> null)",
					"(null < %s)", "(%s < null)",
					"(null > %s)", "(%s > null)",
					"(null >= %s)", "(%s >= null)",
					"(null <= %s)", "(%s <= null)",
				),
				&Expression{})
		case schema.Float:
			if seed.BooleanWithProbability(0.25) {
				return helperclauses.CreateAssembler(
					seed.RandomStringFromChoice("(%s^null)", "(null^%s)"),
					&Expression{Conf: c.Conf},
				)
			}
			fallthrough
		case schema.Integer:
			if seed.BooleanWithProbability(0.2) {
				return helperclauses.CreateStringer("(-null)")
			}
			return helperclauses.CreateAssembler(
				seed.RandomStringFromChoice(
					"(%s+null)", "(null+%s)",
					"(%s-null)", "(null-%s)",
					"(%s*null)", "(null*%s)",
					"(%s/null)", "(null/%s)",
				),
				&Expression{Conf: c.Conf},
			)
		}
	}
	return nil
}
