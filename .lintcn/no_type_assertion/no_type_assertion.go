// lintcn:name no-type-assertion
// lintcn:severity warn
// lintcn:description Flag all type assertions (as X) and show the actual expression type so agents can remove them

package no_type_assertion

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/compiler"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// formatType returns a type string, appending the expanded structural form
// in parentheses when it differs from the alias name. For example:
//
//	"User" → "User ({ name: string; age: number; })"
//	"string" → "string" (no expansion, already concrete)
func formatType(c *checker.Checker, program *compiler.Program, t *checker.Type) string {
	name := c.TypeToString(t)

	// Don't try to expand any/unknown/never — meaningless
	if utils.IsTypeAnyType(t) || utils.IsTypeUnknownType(t) {
		return name
	}
	if utils.IsTypeFlagSet(t, checker.TypeFlagsNever) {
		return name
	}

	// For union types, expand each part individually
	parts := utils.UnionTypeParts(t)
	if len(parts) > 1 {
		anyExpanded := false
		expandedParts := make([]string, len(parts))
		for i, part := range parts {
			expandedParts[i] = formatType(c, program, part)
			if expandedParts[i] != c.TypeToString(part) {
				anyExpanded = true
			}
		}
		if anyExpanded {
			return fmt.Sprintf("%s (%s)", name, strings.Join(expandedParts, " | "))
		}
		return name
	}

	// Build structural form manually from type properties.
	// This expands named types (interfaces, type aliases) into their shape.
	expanded := expandTypeStructure(c, program, t)
	if expanded == "" || expanded == name {
		return name
	}
	return fmt.Sprintf("%s (%s)", name, expanded)
}

// expandTypeStructure builds a structural representation of a type by
// iterating its properties, call signatures, and index signatures.
// Returns "" when the type can't be meaningfully expanded (primitives,
// already-structural types, etc.)
func expandTypeStructure(c *checker.Checker, program *compiler.Program, t *checker.Type) string {
	// Only expand object-like types (interfaces, type aliases, classes)
	if !utils.IsObjectType(t) {
		return ""
	}

	// Skip built-in library types (Array, Map, Promise, Error, etc.)
	// Their structural expansion is just noise (dozens of methods).
	if t.Symbol() != nil && utils.IsSymbolFromDefaultLibrary(program, t.Symbol()) {
		return ""
	}

	props := checker.Checker_getPropertiesOfType(c, t)
	callSigs := c.GetCallSignatures(t)
	indexInfos := checker.Checker_getIndexInfosOfType(c, t)

	// Skip if no structural content to show
	if len(props) == 0 && len(callSigs) == 0 && len(indexInfos) == 0 {
		return ""
	}

	// Skip types with too many members — these are built-in types like Array,
	// Map, Promise, etc. whose full expansion is noise (50+ methods).
	// User-defined types rarely have more than ~20 properties.
	if len(props) > 20 {
		return ""
	}

	var parts []string

	// Index signatures: [key: string]: number
	for _, idx := range indexInfos {
		keyStr := c.TypeToString(idx.KeyType())
		valStr := c.TypeToString(idx.ValueType())
		if idx.IsReadonly() {
			parts = append(parts, fmt.Sprintf("readonly [key: %s]: %s", keyStr, valStr))
		} else {
			parts = append(parts, fmt.Sprintf("[key: %s]: %s", keyStr, valStr))
		}
	}

	// Call signatures: (...args) => return
	for _, sig := range callSigs {
		retType := checker.Checker_getReturnTypeOfSignature(c, sig)
		retStr := c.TypeToString(retType)
		params := checker.Signature_parameters(sig)
		paramParts := make([]string, len(params))
		for i, p := range params {
			pType := checker.Checker_getTypeOfSymbol(c, p)
			paramParts[i] = fmt.Sprintf("%s: %s", p.Name, c.TypeToString(pType))
		}
		parts = append(parts, fmt.Sprintf("(%s) => %s", strings.Join(paramParts, ", "), retStr))
	}

	// Properties: name: type
	for _, prop := range props {
		propType := checker.Checker_getTypeOfSymbol(c, prop)
		propStr := c.TypeToString(propType)
		optional := ""
		if prop.Flags&ast.SymbolFlagsOptional != 0 {
			optional = "?"
		}
		parts = append(parts, fmt.Sprintf("%s%s: %s", prop.Name, optional, propStr))
	}

	return "{ " + strings.Join(parts, "; ") + " }"
}

// unwrapAssertionChain walks back through nested as-expressions to find the
// original expression type before any as-casts. Returns nil if the innermost
// expression is also any/unknown (nothing useful to show).
//
//	(x as unknown) as User → returns type of x
//	(x as unknown as any) as User → returns type of x
//	x as User → returns nil (x is not an as-expression)
func unwrapAssertionChain(ctx rule.RuleContext, expr *ast.Node) *checker.Type {
	inner := ast.SkipParentheses(expr)
	for inner.Kind == ast.KindAsExpression || inner.Kind == ast.KindTypeAssertionExpression {
		inner = ast.SkipParentheses(inner.Expression())
	}
	// If we didn't unwrap anything, there's no chain
	if inner == ast.SkipParentheses(expr) {
		return nil
	}
	t := ctx.TypeChecker.GetTypeAtLocation(inner)
	// If the original is also any/unknown, nothing useful to show
	if utils.IsTypeAnyType(t) || utils.IsTypeUnknownType(t) {
		return nil
	}
	return t
}

var NoTypeAssertionRule = rule.Rule{
	Name: "no-type-assertion",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		checkAssertion := func(node *ast.Node) {
			// Skip `as const` — these are value assertions, not type assertions
			if ast.IsConstAssertion(node) {
				return
			}

			expression := node.Expression()
			typeAnnotation := node.Type()

			expressionType := ctx.TypeChecker.GetTypeAtLocation(expression)
			assertedType := ctx.TypeChecker.GetTypeAtLocation(typeAnnotation)

			assertedStr := formatType(ctx.TypeChecker, ctx.Program, assertedType)

			// When the expression type is any/unknown — walk back through nested
			// as-expressions to find the original source type before the casts.
			// e.g. (x as unknown) as User → show "string" not "unknown"
			if utils.IsTypeAnyType(expressionType) || utils.IsTypeUnknownType(expressionType) {
				originalType := unwrapAssertionChain(ctx, expression)
				if originalType != nil {
					originalStr := formatType(ctx.TypeChecker, ctx.Program, originalType)
					ctx.ReportNode(node, rule.RuleMessage{
						Id: "typeAssertionFromAny",
						Description: fmt.Sprintf(
							"Type assertion `as %s` from `%s`. The original expression type is `%s`. Consider narrowing the type instead.",
							assertedStr, ctx.TypeChecker.TypeToString(expressionType), originalStr,
						),
					})
				} else {
					ctx.ReportNode(node, rule.RuleMessage{
						Id: "typeAssertionFromAny",
						Description: fmt.Sprintf(
							"Type assertion `as %s` from `%s`. Consider adding a type annotation at the source instead.",
							assertedStr, ctx.TypeChecker.TypeToString(expressionType),
						),
					})
				}
				return
			}

			expressionStr := formatType(ctx.TypeChecker, ctx.Program, expressionType)

			// When expression already has the asserted type, it's redundant
			if expressionType == assertedType {
				ctx.ReportNode(node, rule.RuleMessage{
					Id: "typeAssertionRedundant",
					Description: fmt.Sprintf(
						"Type assertion `as %s` is redundant, the expression already has this type. Remove the assertion.",
						ctx.TypeChecker.TypeToString(assertedType),
					),
				})
				return
			}

			// General case: show both the asserted type and the actual expression type
			ctx.ReportNode(node, rule.RuleMessage{
				Id: "typeAssertion",
				Description: fmt.Sprintf(
					"Type assertion `as %s`. The expression type is `%s`. Try removing the assertion or narrowing the type instead.",
					assertedStr, expressionStr,
				),
			})
		}

		return rule.RuleListeners{
			ast.KindAsExpression:            checkAssertion,
			ast.KindTypeAssertionExpression: checkAssertion,
		}
	},
}
