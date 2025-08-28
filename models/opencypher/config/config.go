/*
Package config provides the config for adapt query generation to targets.

Implementations may return their own config to disable
some property types and functions, or provide their own additional functions.
*/
package config

import "github.com/Anon10214/dinkel/models/opencypher/schema"

// The Config for OpenCypher query generation
type Config struct {
	// If this is set, then the targets of SET, DELETE and REMOVE clauses will always be simple
	// variables referencing nodes or relationships instead of general expressions evaluating to
	// nodes or relationships (e.g. through a CASE expression or a startNode function invocation).
	// Setting this also implies that NULL is an invalid candidate.
	OnlyVariablesAsWriteTarget bool
	// If this is set, then any node or edge that has been deleted cannot be referenced again within the query.
	DisallowDeletedWriteTargets bool
	// Whether WITH * is invalid if there are no variables in scope
	AsteriskNeedsTargets bool
	// Integer division may always be inaccurate.
	// This option disallows equivalence transformations such as `x` -> `x/1`
	InaccurateDivision bool
	// These property types won't be generated
	DisallowedPropertyTypes []schema.PropertyType
	// The names of disallowed functions. Function names in this slice won't be generated.
	DisallowedFunctions []string
	// Some implementations don't allow queries such as `MATCH () OPTIONAL MATCH () RETURN 0`,
	// setting this flag disables generation of such queries
	DisallowMatchAfterOptionalMatch bool
	// Functions returning property values in addition to the ones from the OpenCypher specifications.
	//
	// Guaranteed to be non-nil during query generation.
	AdditionalPropertyFunctions map[schema.PropertyType][]schema.Function
	// Functions returning structural values in addition to the ones from the OpenCypher specifications
	//
	// Guaranteed to be non-nil during query generation.
	AdditionalStructuralFunctions map[schema.StructuralType][]schema.Function
	// Aggregation functions in addition to the ones from the OpenCypher specifications
	//
	// Guaranteed to be non-nil during query generation.
	AdditionalAggregationFunctions map[schema.PropertyType][]schema.Function
	// Functions returning maps in addition to the ones from the OpenCypher specifications
	AdditionalMapFunctions []schema.Function
}

// The config currently in use for generation
var usedConfig Config

// SetConfig sets the generation config to be used by all clauses.
func SetConfig(conf Config) {
	// Ensure additional functions are not nil
	if conf.AdditionalPropertyFunctions == nil {
		conf.AdditionalPropertyFunctions = make(map[schema.PropertyType][]schema.Function)
	}
	if conf.AdditionalStructuralFunctions == nil {
		conf.AdditionalStructuralFunctions = make(map[schema.StructuralType][]schema.Function)
	}
	if conf.AdditionalAggregationFunctions == nil {
		conf.AdditionalAggregationFunctions = make(map[schema.PropertyType][]schema.Function)
	}

	usedConfig = conf
}

// GetConfig returns the currently set config.
func GetConfig() Config {
	return usedConfig
}
