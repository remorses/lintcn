// lintcn:name no-unsafe-unknown
// lintcn:severity warn
// lintcn:description Warn on `unknown` in function parameter types and type assertions, with context-specific guidance for safer typing.

// Package no_unsafe_unknown implements a warning rule for `unknown` escape
// hatches in function parameter types and type assertions.
package no_unsafe_unknown

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func safeTypeString(c *checker.Checker, t *checker.Type) string {
	if c == nil || t == nil {
		return "unknown"
	}
	return c.TypeToString(t)
}

func typeNodeContainsUnknown(node *ast.Node, skipNestedParameters bool) bool {
	if node == nil {
		return false
	}
	if node.Kind == ast.KindUnknownKeyword {
		return true
	}
	return node.ForEachChild(func(child *ast.Node) bool {
		if skipNestedParameters && child != nil && child.Kind == ast.KindParameter {
			return false
		}
		return typeNodeContainsUnknown(child, skipNestedParameters)
	})
}

func unwrapAssertionChain(ctx rule.RuleContext, expr *ast.Node) *checker.Type {
	if expr == nil {
		return nil
	}
	inner := ast.SkipParentheses(expr)
	if inner == nil {
		return nil
	}
	start := inner
	for steps := 0; steps < 64; steps++ {
		if inner.Kind != ast.KindAsExpression && inner.Kind != ast.KindTypeAssertionExpression {
			break
		}
		next := inner.Expression()
		if next == nil {
			break
		}
		inner = ast.SkipParentheses(next)
		if inner == nil {
			break
		}
	}
	if inner == start {
		return nil
	}
	t := ctx.TypeChecker.GetTypeAtLocation(inner)
	if t == nil {
		return nil
	}
	if utils.IsTypeAnyType(t) || utils.IsTypeUnknownType(t) {
		return nil
	}
	return t
}

func parameterName(param *ast.ParameterDeclaration) string {
	if param == nil {
		return "parameter"
	}
	name := param.Name()
	if name != nil && ast.IsIdentifier(name) {
		return name.Text()
	}
	return "parameter"
}

func buildUnknownFunctionParameterMessage(paramName string, declaredType string) rule.RuleMessage {
	description := fmt.Sprintf(
		"Function parameter `%s` contains `unknown` in `%s`. Using `unknown` in function arguments to get around TypeScript errors is a code smell. Read the real dependency and value types, and type the parameter correctly instead of silencing the checker with generic helpers or wrappers.",
		paramName,
		declaredType,
	)
	if declaredType == "unknown" {
		description = fmt.Sprintf(
			"Function parameter `%s` uses `unknown`. Using `unknown` in function arguments to get around TypeScript errors is a code smell. Read the real dependency and value types, and type the parameter correctly instead of silencing the checker with generic helpers or wrappers.",
			paramName,
		)
	}
	return rule.RuleMessage{
		Id:          "unknownFunctionParameter",
		Description: description,
	}
}

func buildUnknownAssertionMessage(assertedType string) rule.RuleMessage {
	return rule.RuleMessage{
		Id: "unknownAssertion",
		Description: fmt.Sprintf(
			"Type assertion to `%s` uses `unknown`. This is usually a forced conversion hiding a real type mismatch. Read the actual source and target types, then narrow or transform the value instead of laundering it through `unknown`.",
			assertedType,
		),
	}
}

func buildUnknownAssertionFromUnknownMessage(assertedType string, originalType string) rule.RuleMessage {
	description := fmt.Sprintf(
		"Type assertion to `%s` from `unknown`. This usually means the code is trying to force an incompatible conversion. Narrow the value with real runtime checks or fix the source types instead of forcing it through `unknown`.",
		assertedType,
	)
	if originalType != "" {
		description = fmt.Sprintf(
			"Type assertion to `%s` from `unknown`. The original expression type is `%s`. Using `unknown` in an assertion to force a different type is a serious code smell. Read the real types and narrow or transform the value instead.",
			assertedType,
			originalType,
		)
	}
	return rule.RuleMessage{
		Id:          "unknownAssertionFromUnknown",
		Description: description,
	}
}

var NoUnsafeUnknownRule = rule.Rule{
	Name: "no-unsafe-unknown",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		checkParameter := func(node *ast.Node) {
			if node == nil || ast.IsThisParameter(node) {
				return
			}
			param := node.AsParameterDeclaration()
			if param == nil || param.Type == nil {
				return
			}
			if !typeNodeContainsUnknown(param.Type, true) {
				return
			}

			declaredType := checker.Checker_getTypeFromTypeNode(ctx.TypeChecker, param.Type)
			ctx.ReportNode(param.Type, buildUnknownFunctionParameterMessage(
				parameterName(param),
				safeTypeString(ctx.TypeChecker, declaredType),
			))
		}

		checkAssertion := func(node *ast.Node) {
			if node == nil || ast.IsConstAssertion(node) {
				return
			}
			expression := node.Expression()
			typeAnnotation := node.Type()
			if expression == nil || typeAnnotation == nil {
				return
			}

			assertedType := checker.Checker_getTypeFromTypeNode(ctx.TypeChecker, typeAnnotation)
			assertedTypeStr := safeTypeString(ctx.TypeChecker, assertedType)

			if typeNodeContainsUnknown(typeAnnotation, false) {
				ctx.ReportNode(node, buildUnknownAssertionMessage(assertedTypeStr))
				return
			}

			expressionType := ctx.TypeChecker.GetTypeAtLocation(expression)
			if expressionType == nil || !utils.IsTypeUnknownType(expressionType) {
				return
			}

			originalType := ""
			if unwrapped := unwrapAssertionChain(ctx, expression); unwrapped != nil {
				originalType = safeTypeString(ctx.TypeChecker, unwrapped)
			}
			ctx.ReportNode(node, buildUnknownAssertionFromUnknownMessage(assertedTypeStr, originalType))
		}

		return rule.RuleListeners{
			ast.KindParameter:               checkParameter,
			ast.KindAsExpression:            checkAssertion,
			ast.KindTypeAssertionExpression: checkAssertion,
		}
	},
}
