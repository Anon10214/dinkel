package clauses

import (
	"fmt"
	"math"

	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/sirupsen/logrus"
)

func generatePropertyType(seed *seed.Seed) schema.PropertyType {
	conf := config.GetConfig()
	for {
		// Generate a property type, excluding ANY_CONSTANT
		genType := schema.PropertyType(seed.GetRandomIntn(12) + 1)
		// Make sure the property type isn't disallowed
		var isDisallowed bool
		for _, ignored := range conf.DisallowedPropertyTypes {
			if ignored == genType {
				isDisallowed = true
				break
			}
		}
		if !isDisallowed {
			return genType
		}
	}
}

// TODO: Add more types
// Generates a random literal. A literal cannot be a composite type like a list
func generateLiteral(seed *seed.Seed, targetType schema.PropertyType, mustBeNonNull bool) string {
	// Get a concrete type
	if targetType == schema.AnyType {
		targetType = generatePropertyType(seed)
	}

	if seed.RandomBoolean() {
		return generateInterestingLiteral(seed, targetType, mustBeNonNull)
	}

	// Normalize it first
	switch targetType {
	case schema.Boolean:
		if seed.RandomBoolean() {
			return "true"
		}
		return "false"
	case schema.Date:
		return "date('2000-01-01')"
	case schema.Datetime:
		return "datetime('2015-06-24T12:50:35.556+0100')"
	case schema.Duration:
		return `duration("P1Y")`
	case schema.Float:
		return fmt.Sprint(math.Float64frombits(uint64(seed.GetRandomInt64())))
	case schema.Integer:
		return fmt.Sprint(seed.GetRandomInt64())
	case schema.LocalDateTime:
		return "localdatetime('2015185T19:32:24')"
	case schema.LocalTime:
		return "localtime('12:50:35.556')"
	case schema.Point:
		return fmt.Sprintf("point({x: %d, y: %d})", seed.GetRandomInt64(), seed.GetRandomInt64())
	case schema.String:
		return `"ABC"`
	case schema.Time:
		return "time('125035.556+0100')"
	case schema.PositiveInteger:
		return fmt.Sprint(seed.GetRandomPositiveInt64())
	case schema.Percentile: // Float in [0, 1]
		return fmt.Sprint(math.Abs(math.Remainder(math.Float64frombits(uint64(seed.GetRandomPositiveInt64())), 1.0)))
	case schema.Int32: // Int in [-2147483648, 2147483647]
		return fmt.Sprint(int32(seed.GetRandomInt64()))
	case schema.PositiveInt32: // Int in [0, 2147483647]
		return fmt.Sprint(seed.GetRandomPositiveInt64() % 2147483648)
	}
	logrus.Errorf("Called generateLiteral with invalid targetType: %d", targetType)
	return ""
}

// TODO: Add more types
// Generates an "interesting" property value (mainly edge case constants like 0 or 1, MAX_INT, MIN_INT etc)
func generateInterestingLiteral(seed *seed.Seed, targetType schema.PropertyType, mustBeNonNull bool) string {
	if !mustBeNonNull && seed.BooleanWithProbability(0.25) {
		return "null"
	}
	switch targetType {
	case schema.Float:
		return seed.RandomStringFromChoice(
			"0.0",
			"-0.0",
			"1.0",
			"-1.0",
			// Java Double.MAX_VALUE
			"1.7976931348623157E308",
			// Java -Double.MAX_VALUE
			"-1.7976931348623157E308",
			// Java Double.MIN_VALUE
			"4.9E-324",
			// Java -Double.MIN_VALUE
			"-4.9E-324",
		)
	case schema.Integer:
		return seed.RandomStringFromChoice(
			"0",
			"1",
			"-1",
			// Java Long.MAX_VALUE
			"9223372036854775807",
			// Java Long.MIN_VALUE
			"-9223372036854775808",
		)
	case schema.PositiveInteger:
		return seed.RandomStringFromChoice(
			"0",
			"1",
			// Java Long.MAX_VALUE
			"9223372036854775807",
		)
	case schema.Percentile:
		return seed.RandomStringFromChoice(
			"0.0",
			"1.0",
		)
	case schema.Int32:
		return seed.RandomStringFromChoice(
			"0",
			"1",
			"-1",
			// Java Int.MAX_VALUE
			"2147483647",
			// Java Int.MIN_VALUE
			"-2147483648",
		)
	case schema.PositiveInt32:
		return seed.RandomStringFromChoice(
			"0",
			"1",
			// Java Int.MAX_VALUE
			"2147483647",
		)
	}
	return generateLiteral(seed, targetType, mustBeNonNull)
}