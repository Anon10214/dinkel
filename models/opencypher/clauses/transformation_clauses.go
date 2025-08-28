package clauses

// This file contains clauses that can be used during equvalence transformations.
// They guarantee certain semantics but due to their more rigid structure are not useful for general query generation.

import (
	"fmt"

	"github.com/Anon10214/dinkel/models/opencypher/config"
	"github.com/Anon10214/dinkel/models/opencypher/schema"
	"github.com/Anon10214/dinkel/seed"
	"github.com/Anon10214/dinkel/translator"
	"github.com/Anon10214/dinkel/translator/helperclauses"
)

// Fully generates an expression's AST
func generateExpressionAST(conf schema.ExpressionConfig, seed *seed.Seed, schema *schema.Schema) *helperclauses.ClauseCapturer {
	clause := &Expression{Conf: conf}
	capturer := helperclauses.GetClauseCapturerForClause(clause)
	capturer.GenerateAST(seed, schema)
	return capturer
}

// Tautum always evaluates to true
type Tautum struct {
	conf schema.ExpressionConfig
}

func (c *Tautum) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	switch seed.GetRandomIntn(4) {
	case 0:
		return []translator.Clause{&TautumPartition{conf: c.conf}}
	case 1:
		return []translator.Clause{helperclauses.CreateAssembler(
			seed.RandomStringFromChoice("((%s) and (%s))", "((%s) or (%s))"),
			&Tautum{conf: c.conf}, &Tautum{conf: c.conf},
		)}
	case 2:
		return []translator.Clause{helperclauses.CreateAssembler("(NOT (%s))", &Falsum{conf: c.conf})}
	}
	return []translator.Clause{helperclauses.CreateStringer("true")}
}

// Falsum always evaluates to false
type Falsum struct {
	conf schema.ExpressionConfig
}

func (c *Falsum) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	switch seed.GetRandomIntn(4) {
	case 0:
		return []translator.Clause{&FalsumPartition{conf: c.conf}}
	case 1:
		return []translator.Clause{helperclauses.CreateAssembler(
			seed.RandomStringFromChoice("((%s) and (%s))", "((%s) or (%s))"),
			&Falsum{conf: c.conf}, &Falsum{conf: c.conf},
		)}
	case 2:
		return []translator.Clause{helperclauses.CreateAssembler("(NOT (%s))", &Tautum{conf: c.conf})}
	}
	return []translator.Clause{helperclauses.CreateStringer("false")}
}

type DeadCode struct {
	oldAllowOnlyNonNullPropertyExpressions bool
}

func (c *DeadCode) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	s.CannotReturn = true
	c.oldAllowOnlyNonNullPropertyExpressions = s.IsInMergeClause

	if seed.BooleanWithProbability(0.9) {
		// With high probability, choose an element that is unlikely to explode the AST
		if seed.RandomBoolean() {
			// SET <x> += {}, <y> += {}, ...
			nodeVariables := s.StructuralVariablesByType[schema.NODE]
			relationShipVariables := s.StructuralVariablesByType[schema.RELATIONSHIP]
			availableVariables := append(nodeVariables, relationShipVariables...)
			// Can only generate if there is an available variable
			if len(availableVariables) > 0 {
				templStr := fmt.Sprintf("SET %s += {}", availableVariables[seed.GetRandomIntn(len(availableVariables))].Name)
				for seed.RandomBoolean() {
					templStr += fmt.Sprintf(", %s += {}", availableVariables[seed.GetRandomIntn(len(availableVariables))].Name)
				}
				return []translator.Clause{helperclauses.CreateStringer(templStr), optionalClause(seed, helperclauses.CreateAssembler(" WITH * %s", &DeadCode{}))}
			}
		}
		// MATCH <x> MERGE <x> [ON CREATE <y>]
		s.IsInMergeClause = true
		separatorName := generateUniqueName(seed, s)
		createElem := helperclauses.GetClauseCapturerForClause(&CreateElement{})
		createElem.GenerateAST(seed, s)
		vars := ""
		for v := range s.PropertyVariablesByName {
			vars += v + ", "
		}
		for v := range s.StructuralVariablesByName {
			vars += v + ", "
		}

		var setExpr translator.Clause = &EmptyClause{}
		// Can generate expression if the config allows for non-variables as write targets, or else if variables are available
		canGenerateSetExpression := !config.GetConfig().OnlyVariablesAsWriteTarget || len(s.StructuralVariablesByType[schema.RELATIONSHIP])+len(s.StructuralVariablesByType[schema.NODE]) != 0
		if canGenerateSetExpression {
			setExpr = optionalClause(seed, helperclauses.CreateAssembler("ON CREATE SET %s", &SetExpression{}))
		}

		return []translator.Clause{helperclauses.CreateAssembler(
			"MATCH %s %s WITH %s0 AS %s MERGE %s %s %s",
			createElem,
			&OptionalWhereClause{},
			helperclauses.CreateStringer(vars),
			helperclauses.CreateStringer(separatorName),
			createElem.Copy(),
			setExpr,
			optionalClauseWithProbability(seed, helperclauses.CreateAssembler("WITH * %s", &DeadCode{}), 0.1),
		)}
	}

	if seed.RandomBoolean() {
		// MATCH <x> WHERE <falsum>
		return []translator.Clause{helperclauses.CreateAssembler("MATCH %s WHERE %s %s", &MatchElementChain{}, &Falsum{}, &ReadClause{})}
	}

	s.UseNewLabelMatchType = new(bool)

	// Create and generate previous elements of the match clause to make sure we have access to the newest variables
	previousMatchChain := helperclauses.GetClauseCapturerForClause(optionalClause(seed, helperclauses.CreateAssembler("%s, ", &MatchElementChain{})))
	previousMatchChain.GenerateAST(seed, s)
	previousPathPattern := helperclauses.GetClauseCapturerForClause(optionalClause(seed, &PathPatternExpression{}))
	previousPathPattern.GenerateAST(seed, s)

	subclauses := []translator.Clause{helperclauses.CreateStringer("MATCH "), previousMatchChain, previousPathPattern}

	if _, ok := previousPathPattern.GetCapturedClause().(*EmptyClause); !ok {
		// If we actually generated a path pattern, have to connect them through a relationship
		relationship := helperclauses.GetClauseCapturerForClause(helperclauses.CreateAssembler(seed.RandomStringFromChoice("<-%s-", "-%s-", "-%s->"), &MatchRelationship{}))
		relationship.GenerateAST(seed, s)

		subclauses = append(subclauses, relationship)
	}

	subclauses = append(subclauses, &NonexistantPattern{})

	// Optionally expand the match with more match elements
	subclauses = append(subclauses, optionalClause(seed, helperclauses.CreateAssembler(", %s", &MatchElementChain{})))

	subclauses = append(subclauses, &ReadClause{})
	return subclauses
}

func (c DeadCode) ModifySchema(s *schema.Schema) {
	s.IsInMergeClause = c.oldAllowOnlyNonNullPropertyExpressions
}

// NonexistantPattern generates a pattern that cannot match any graph elements
type NonexistantPattern struct{}

func (c *NonexistantPattern) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	// MATCH <x>; Where x contains at least one new property or label
	newIdent := generateUniqueName(seed, s)

	subclauses := []translator.Clause{}
	decideOnLabelMatchType(seed, s)

	if seed.RandomBoolean() {
		// Add new label
		s.Labels[schema.ANY] = append(s.Labels[schema.ANY], newIdent)
		if seed.RandomBoolean() {
			// Add new label to a node
			s.Labels[schema.NODE] = append(s.Labels[schema.NODE], newIdent)

			// Append the node match
			subclauses = append(subclauses, helperclauses.CreateAssembler("(%s%s%s%s%s)",
				getOptionalStructureName(schema.NODE, false),
				&OptionalLabelMatch{LabelType: schema.NODE},
				helperclauses.CreateStringer(":"+newIdent),
				&OptionalLabelMatch{LabelType: schema.NODE},
				&OptionalPropertyMatch{},
			))

			// Optionally expand the path pattern
			nextRelationship := helperclauses.CreateAssembler(seed.RandomStringFromChoice("<-%s-", "-%s-", "-%s->"), &MatchRelationship{})
			subclauses = append(subclauses, optionalClause(seed, helperclauses.CreateAssemblerWithoutTemplateString(nextRelationship, &PathPatternExpression{})))
		} else {
			// Add new label to a relationship
			s.Labels[schema.RELATIONSHIP] = append(s.Labels[schema.RELATIONSHIP], newIdent)

			subclauses = append(subclauses, &PathPatternExpression{})

			relationshipTemplString := seed.RandomStringFromChoice("<-[%s:%s%s]-", "-[%s:%s%s]-", "-[%s:%s%s]->")

			subclauses = append(subclauses, helperclauses.CreateAssembler(relationshipTemplString,
				getOptionalStructureName(schema.RELATIONSHIP, false),
				helperclauses.CreateStringer(newIdent),
				&OptionalPropertyMatch{},
			))

			subclauses = append(subclauses, &PathPatternExpression{})
		}
	} else {
		// Add new property
		newPropertyType := generatePropertyType(seed)

		s.AddProperty(schema.Property{Name: newIdent, Type: newPropertyType})
		if seed.RandomBoolean() {
			// Add new property to a node

			// Append the node match
			subclauses = append(subclauses, helperclauses.CreateAssembler("(%s%s{%s%s%s%s})",
				getOptionalStructureName(schema.NODE, false),
				&OptionalLabelMatch{LabelType: schema.NODE},
				optionalClause(seed, helperclauses.CreateAssembler("%s, ", &PropertyChain{CreateNewPropertyProbability: 0.05})),
				helperclauses.CreateStringer(newIdent+":"),
				&Expression{Conf: schema.ExpressionConfig{TargetType: schema.PropertyValue, PropertyType: newPropertyType, MustBeNonNull: s.IsInMergeClause}},
				optionalClause(seed, helperclauses.CreateAssembler(", %s", &PropertyChain{CreateNewPropertyProbability: 0.05})),
			))

			// Optionally expand the path pattern
			nextRelationship := helperclauses.CreateAssembler(seed.RandomStringFromChoice("<-%s-", "-%s-", "-%s->"), &MatchRelationship{})
			subclauses = append(subclauses, optionalClause(seed, helperclauses.CreateAssemblerWithoutTemplateString(nextRelationship, &PathPatternExpression{})))
		} else {
			// Add new property to a relationship
			subclauses = append(subclauses, &PathPatternExpression{})

			relationshipTemplString := seed.RandomStringFromChoice("<-[%s:%s{%s%s%s%s}]-", "-[%s:%s{%s%s%s%s}]-", "-[%s:%s{%s%s%s%s}]->")

			subclauses = append(subclauses, helperclauses.CreateAssembler(relationshipTemplString,
				getOptionalStructureName(schema.RELATIONSHIP, false),
				&LabelMatch{LabelType: schema.RELATIONSHIP},
				optionalClause(seed, helperclauses.CreateAssembler("%s, ", &PropertyChain{CreateNewPropertyProbability: 0.05})),
				helperclauses.CreateStringer(newIdent+":"),
				&Expression{Conf: schema.ExpressionConfig{TargetType: schema.PropertyValue, PropertyType: newPropertyType, MustBeNonNull: s.IsInMergeClause}},
				optionalClause(seed, helperclauses.CreateAssembler(", %s", &PropertyChain{CreateNewPropertyProbability: 0.05})),
			))

			subclauses = append(subclauses, &PathPatternExpression{})
		}
	}

	return subclauses
}

// Tautum partition generates an expression of the type
//
//	(P) or (not P) or (P IS NULL)
type TautumPartition struct {
	conf schema.ExpressionConfig
}

func (c *TautumPartition) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.conf.TargetType = schema.PropertyValue
	c.conf.PropertyType = schema.Boolean
	c.conf.IsList = false
	expr := generateExpressionAST(c.conf, seed, s)
	return []translator.Clause{
		helperclauses.CreateStringer("("),
		optionalClause(seed, helperclauses.CreateAssembler("(%s) or ", &Expression{Conf: c.conf})),
		helperclauses.CreateAssembler(" (%s) ", expr.Copy()),
		optionalClause(seed, helperclauses.CreateAssembler(" or (%s) ", &Expression{Conf: c.conf})),
		helperclauses.CreateAssembler(" or (NOT %s) ", expr.Copy()),
		optionalClause(seed, helperclauses.CreateAssembler(" or (%s) ", &Expression{Conf: c.conf})),
		helperclauses.CreateAssembler(" or ((%s) IS NULL)", expr.Copy()),
		optionalClause(seed, helperclauses.CreateAssembler(" or (%s)", &Expression{Conf: c.conf})),
		helperclauses.CreateStringer(")"),
	}
}

// Falsum partition generates an expression of the type
//
//	(P) and (not P) and (P IS NOT NULL)
type FalsumPartition struct {
	conf schema.ExpressionConfig
}

func (c *FalsumPartition) Generate(seed *seed.Seed, s *schema.Schema) []translator.Clause {
	c.conf.TargetType = schema.PropertyValue
	c.conf.PropertyType = schema.Boolean
	c.conf.IsList = false
	expr := generateExpressionAST(c.conf, seed, s)
	return []translator.Clause{
		helperclauses.CreateStringer("("),
		optionalClause(seed, helperclauses.CreateAssembler("(%s) and ", &Expression{Conf: c.conf})),
		helperclauses.CreateAssembler(" (%s) ", expr.Copy()),
		optionalClause(seed, helperclauses.CreateAssembler(" and (%s) ", &Expression{Conf: c.conf})),
		helperclauses.CreateAssembler(" and (NOT %s) ", expr.Copy()),
		optionalClause(seed, helperclauses.CreateAssembler(" and (%s) ", &Expression{Conf: c.conf})),
		helperclauses.CreateAssembler(" and ((%s) IS NOT NULL)", expr.Copy()),
		optionalClause(seed, helperclauses.CreateAssembler(" and (%s)", &Expression{Conf: c.conf})),
		helperclauses.CreateStringer(")"),
	}
}
