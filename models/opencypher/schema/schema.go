/*
Package schema defines the schema used for complex, stateful query generation.

The schema holds the current query context as well as an approximation
of the current DB state. It holds information like the current variables as well
as their types and graph elements like properties, labels and relationship types.

Additionally, it holds information about certain important values, which
ensure syntactic correctness (e.g. IsUnionAll).

The schema provides some helper functions to ensure that values are correctly
inserted and kept track of.
*/
package schema

import (
	"github.com/qdm12/reprint"
	"github.com/sirupsen/logrus"
)

// The Schema used for stateful query generation
type Schema struct {
	// Lists the names of all properties used in a graph element
	Properties         map[PropertyType][]Property
	PropertyTypeByName map[string]PropertyType
	// Lists all labels used for a graph element
	Labels map[StructuralType][]string
	// Whether the statement already has an OPTIONAL MATCH clause (disallows further use of normal MATCH clauses)
	// TODO: Not every DBMS needs this feature, some accept MATCH after OPTIONAL MATCH
	HasOptionalMatch bool

	// Added because of https://github.com/neo4j/neo4j/issues/13054
	IsInSubquery bool

	// If this is set, the query will no longer generate any write clauses
	DisallowWriteClauses bool

	// If this is set, no RETURN clause can be generated
	CannotReturn bool

	// There are two label match types in neo4j, the old one (only allowing ANDing labels by separating them with a colon)
	// And the new one, allowing complex expressions with negation, ORing and ANDing labels, plus adding wildcards
	// According to [Neo4j Docs], every clause must use the same label match type
	// If this value is unset, it is not yet decided which type to use for this clause
	// The RootClause and WriteQuery must reset this value to nil during generation
	//
	// [Neo4j Docs]: https://neo4j.com/docs/cypher-manual/current/syntax/expressions/#syntax-restrictions-label
	UseNewLabelMatchType *bool

	// If set, this decides whether the unions should be UNION ALL clauses or just UNION clauses
	IsUnionAll *bool

	// If true, expressions are not allowed to evaluate to NULL or contain subquery expressions (COUNT/EXISTS/COLLECT)
	IsInMergeClause bool

	// If true, expressions are not allowed to contain aggregate functions
	DisallowAggregateFunctions bool
	// If this is set, the clause `RETURN *` will never be generated
	DisallowReturnAll bool

	// Map of all used names in the query, used to ensure their uniqueness
	UsedNames *map[string]bool

	// Map of all deleted edge/node variables
	DeletedVars map[string]bool

	// Allows UNION clauses and having CALL subqueries return variables
	MustReturn                  bool // If MustReturn is true, the statement has to terminate with a RETURN clause
	PropertyVariablesToReturn   []PropertyVariable
	StructuralVariablesToReturn []StructuralVariable

	// Property variables (ints, floats, strings, etc) hold variables created using WITH or UNWIND statements and can be used everywhere
	PropertyVariablesByName map[string]PropertyVariable         // Holds property variables, searchable via name
	PropertyVariablesByType map[PropertyType][]PropertyVariable // Holds property variables, collected via type

	// Structural variables (nodes, relationships and paths) can only be used in RETURN and WHERE statements
	StructuralVariablesByName map[string]StructuralVariable           // Holds structural variables, searchable via name
	StructuralVariablesByType map[StructuralType][]StructuralVariable // Holds structural variables, collected via type

	// Holds structural variables that just got created in a CREATE clause, which can't be used in functions in the same clause
	JustCreatedStructuralVariables []StructuralVariable

	// Names of created indexes
	Indexes []string
}

// Reset sets the schema back to an initial state.
//
// It should only be called by the driver, not during clause generation.
// For creating a new context, use NewContext() instead.
func (s *Schema) Reset() {
	s.Labels = make(map[StructuralType][]string)
	s.Properties = make(map[PropertyType][]Property)
	s.PropertyTypeByName = make(map[string]PropertyType)
	usedNames := make(map[string]bool)
	s.UsedNames = &usedNames
	s.DeletedVars = make(map[string]bool)
	s.PropertyVariablesByName = make(map[string]PropertyVariable)
	s.PropertyVariablesByType = make(map[PropertyType][]PropertyVariable)
	s.StructuralVariablesByName = make(map[string]StructuralVariable)
	s.StructuralVariablesByType = make(map[StructuralType][]StructuralVariable)

	s.IsInSubquery = false
}

// Copy returns a deep copy of the schema
func (s Schema) Copy() *Schema {
	copy := reprint.This(s).(Schema)
	// Used names must persist
	copy.UsedNames = s.UsedNames
	return &copy
}

// ResetContext resets the schema's context by resetting fields populated during generation
func (s *Schema) ResetContext() {
	s.HasOptionalMatch = false

	s.PropertyVariablesByName = make(map[string]PropertyVariable)
	s.PropertyVariablesByType = make(map[PropertyType][]PropertyVariable)

	s.DeletedVars = make(map[string]bool)

	s.StructuralVariablesByName = make(map[string]StructuralVariable)
	s.StructuralVariablesByType = make(map[StructuralType][]StructuralVariable)
}

// AddPropertyVariable adds a property variable to the schema.
func (s *Schema) AddPropertyVariable(variable PropertyVariable) {
	var listMask PropertyType
	if variable.Type&PropertyType(ListMask) != 0 {
		listMask = PropertyType(ListMask)
	}

	if variable.Type != AnyType|listMask {
		s.PropertyVariablesByType[AnyType|listMask] = append(s.PropertyVariablesByType[AnyType|listMask], variable)
	}
	s.PropertyVariablesByType[variable.Type] = append(s.PropertyVariablesByType[variable.Type], variable)

	if variable.Type&PropertyType(NullableMask) != 0 {
		nullableType := ^PropertyType(NullableMask) & variable.Type
		s.PropertyVariablesByType[nullableType] = append(s.PropertyVariablesByType[nullableType], variable)
		if nullableType != AnyType|listMask {
			s.PropertyVariablesByType[AnyType|listMask|PropertyType(NullableMask)] = append(s.PropertyVariablesByType[AnyType|listMask|PropertyType(NullableMask)], variable)
		}
	}
	logrus.Tracef("Added property variable with name %s and type 0x%X", variable.Name, variable.Type)

	s.PropertyVariablesByName[variable.Name] = variable
}

// AddStructuralVariable adds a structural variable to the schema.
func (s *Schema) AddStructuralVariable(variable StructuralVariable) {
	var listMask StructuralType
	if variable.Type&StructuralType(ListMask) != 0 {
		listMask = StructuralType(ListMask)
	}

	if variable.Type != ANY|listMask {
		s.StructuralVariablesByType[ANY|listMask] = append(s.StructuralVariablesByType[ANY|listMask], variable)
	}
	s.StructuralVariablesByType[variable.Type] = append(s.StructuralVariablesByType[variable.Type], variable)

	if variable.Type|StructuralType(NullableMask) != 0 {
		nullableType := ^StructuralType(NullableMask) & variable.Type
		s.StructuralVariablesByType[nullableType] = append(s.StructuralVariablesByType[nullableType], variable)
		if nullableType != ANY|listMask {
			s.StructuralVariablesByType[ANY|listMask|StructuralType(NullableMask)] = append(s.StructuralVariablesByType[ANY|listMask|StructuralType(NullableMask)], variable)
		}
	}
	logrus.Tracef("Added structural variable with name %s and type 0x%X", variable.Name, variable.Type)

	s.StructuralVariablesByName[variable.Name] = variable
}

// NewContext creates and returns a new context by resetting the query context but preserving the abstract graph state.
func (s Schema) NewContext() *Schema {
	newSchema := Schema{}
	newSchema.Reset()

	newSchema.DisallowWriteClauses = s.DisallowWriteClauses

	newSchema.Properties = s.Properties
	newSchema.PropertyTypeByName = s.PropertyTypeByName
	newSchema.Labels = s.Labels

	newSchema.UsedNames = s.UsedNames

	newSchema.UseNewLabelMatchType = s.UseNewLabelMatchType
	newSchema.IsInSubquery = s.IsInSubquery
	newSchema.IsInMergeClause = s.IsInMergeClause
	newSchema.DisallowAggregateFunctions = s.DisallowAggregateFunctions
	newSchema.DisallowReturnAll = s.DisallowReturnAll

	return &newSchema
}

// NewSubContext creates and returns a schema which is a subcontext relative to the given schema.
// It keeps variables and relevant flags.
func (s Schema) NewSubContext() *Schema {
	deletedVars := make(map[string]bool)
	for key := range s.DeletedVars {
		deletedVars[key] = true
	}

	propertyVariablesByName := make(map[string]PropertyVariable)
	for id, v := range s.PropertyVariablesByName {
		propertyVariablesByName[id] = v
	}

	propertyVariablesByType := make(map[PropertyType][]PropertyVariable)
	for id, v := range s.PropertyVariablesByType {
		propertyVariablesByType[id] = make([]PropertyVariable, len(v))
		copy(propertyVariablesByType[id], v)
	}

	structuralVariablesByName := make(map[string]StructuralVariable)
	for id, v := range s.StructuralVariablesByName {
		structuralVariablesByName[id] = v
	}

	structuralVariablesByType := make(map[StructuralType][]StructuralVariable)
	for id, v := range s.StructuralVariablesByType {
		structuralVariablesByType[id] = make([]StructuralVariable, len(v))
		copy(structuralVariablesByType[id], v)
	}

	return &Schema{
		Properties:         s.Properties,
		PropertyTypeByName: s.PropertyTypeByName,
		Labels:             s.Labels,

		UsedNames: s.UsedNames,

		UseNewLabelMatchType: s.UseNewLabelMatchType,
		IsInSubquery:         s.IsInSubquery,
		IsInMergeClause:      s.IsInMergeClause,

		DisallowAggregateFunctions: s.DisallowAggregateFunctions,
		DisallowReturnAll:          s.DisallowReturnAll,

		DeletedVars: deletedVars,

		PropertyVariablesByName: propertyVariablesByName,
		PropertyVariablesByType: propertyVariablesByType,

		StructuralVariablesByName: structuralVariablesByName,
		StructuralVariablesByType: structuralVariablesByType,
	}
}

// AddProperty adds a new (node or relationship) property to the abstract graph state
func (s *Schema) AddProperty(property Property) {
	if propType, found := s.PropertyTypeByName[property.Name]; found {
		// Multiple possible types, change it to ANY_CONSTANT
		if propType != property.Type {
			delete(s.PropertyTypeByName, property.Name)
			s.PropertyTypeByName[property.Name] = AnyType
		}
	} else {
		s.PropertyTypeByName[property.Name] = property.Type
	}

	s.Properties[property.Type] = append(s.Properties[property.Type], property)
	s.Properties[AnyType] = append(s.Properties[AnyType], property)
	s.Properties[property.Type] = append(s.Properties[property.Type], property)
}

// Property represents a node's or relationship's property
type Property struct {
	Name  string
	Type  PropertyType
	Value string
}

// StructuralType specifies the exact type of a structural variable.
type StructuralType int

// Structural Types
const (
	ANY StructuralType = iota
	NODE
	RELATIONSHIP
	PATH
)

// A PropertyVariable represents any variable evaluating to a property value.
type PropertyVariable struct {
	Name  string
	Type  PropertyType
	Value string
}

// A StructuralVariable represents any variable evaluating to a structural value.
type StructuralVariable struct {
	Name string
	Type StructuralType
	// If this variable is likely to evaluate to nil.
	// If unset, then the variable is guaranteed to be non nil.
	LikelyNull bool
}

// PropertyType represents a type for a Cypher expression
// If the 16th bit is set, the expression can NOT be null
// If the 15th bit is set, the expression is a list of the underlying type
type PropertyType int

// Masks for property types
const (
	// If this mask is used, the expression can NOT be null
	NullableMask int = 0x8000
	ListMask     int = 0x4000
)

const (
	// AnyType indicates the expression can evaluate to any type
	AnyType PropertyType = iota
)

// Types defined by OpenCypher
const (
	Boolean PropertyType = iota + 1
	Date
	Datetime
	Duration
	Float
	Integer
	LocalDateTime
	LocalTime
	Point
	String
	Time
)

// Types used for more accurate generation
const (
	// For LIMIT & SKIP
	PositiveInteger PropertyType = iota + 12
	// For percentileCont and percentileDisc functions
	Percentile
	// For substring
	Int32
	// For round function precision
	PositiveInt32
)

// The ExpressionType dictates whether an expression evaluates
// to a property or structural value.
type ExpressionType int

const (
	// AnyExpression indicates the expression can evaluate to any expression type
	AnyExpression ExpressionType = iota
	// PropertyValue indicates the expression can only evaluate to a property value
	PropertyValue
	// StructuralValue indicates the expression can only evaluate to a structural value
	StructuralValue
)

// The ExpressionConfig holds all options dictating how an expression gets generated.
type ExpressionConfig struct {
	// If this expression cannot evaluate to null
	MustBeNonNull bool
	// If this expression represents a list of the underlying type
	IsList bool
	// The type of this expression
	TargetType     ExpressionType
	PropertyType   PropertyType   // Only relevant if targetType != STRUCTURAL
	StructuralType StructuralType // Only relevant if targetType != PROPERTY
	// Constant expressions aren't allowed to contain variables
	IsConstantExpression bool
	// If this is expression is allowed to contain aggregating functions, for example in a RETURN or WITH statement
	CanContainAggregatingFunctions bool
	// If this expression is allowed to evaluate to a map
	AllowMaps bool
	// If true, then whatever this expression will evaluate to, will be deleted from the graph
	GetsDeleted bool
}

// The Function struct represents a function callable in a cypher query.
// Its return type gets defined in the implementation-specific OpenCypher config.
type Function struct {
	// The function's name
	Name string
	// The expression configs for the function arguments
	InputTypes []ExpressionConfig
	// If true, this function can always return null even if arguments are all non null
	CanAlwaysBeNull bool
}
