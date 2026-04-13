// lintcn:name prefer-is-truthy
// lintcn:description Prefer a reusable `isTruthy` helper over inline nullable type guards in `filter` callbacks.
//
// Package prefer_is_truthy flags ad hoc inline type guards that only remove
// nullish values inside `filter` callbacks.
package prefer_is_truthy

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func isInlineFilterCallback(node *ast.Node) bool {
	if node == nil || node.Parent == nil || !ast.IsCallExpression(node.Parent) {
		return false
	}

	call := node.Parent.AsCallExpression()
	if call == nil || call.Arguments == nil || !ast.IsPropertyAccessExpression(call.Expression) {
		return false
	}

	isArgument := false
	for _, argument := range call.Arguments.Nodes {
		if argument == node {
			isArgument = true
			break
		}
	}
	if !isArgument {
		return false
	}

	name := call.Expression.AsPropertyAccessExpression().Name()
	return name != nil && name.Text() == "filter"
}

func getTypePredicate(node *ast.Node) *ast.TypePredicateNode {
	if node == nil || node.Type() == nil || !ast.IsTypePredicateNode(node.Type()) {
		return nil
	}

	predicate := node.Type().AsTypePredicateNode()
	if predicate == nil || predicate.AssertsModifier != nil || predicate.ParameterName == nil || predicate.Type == nil {
		return nil
	}
	if !ast.IsIdentifier(predicate.ParameterName) {
		return nil
	}

	return predicate
}

func findPredicateParameter(node *ast.Node, predicate *ast.TypePredicateNode) *ast.ParameterDeclarationNode {
	if node == nil || predicate == nil || predicate.ParameterName == nil || !ast.IsIdentifier(predicate.ParameterName) {
		return nil
	}

	predicateName := predicate.ParameterName.AsIdentifier().Text
	for _, parameter := range node.Parameters() {
		if parameter == nil || parameter.Name() == nil || !ast.IsIdentifier(parameter.Name()) {
			continue
		}
		if parameter.Name().AsIdentifier().Text == predicateName {
			return parameter
		}
	}

	return nil
}

func getReturnedExpression(node *ast.Node) *ast.Node {
	if node == nil || node.Body() == nil {
		return nil
	}

	body := ast.SkipParentheses(node.Body())
	if body == nil {
		return nil
	}
	if !ast.IsBlock(body) {
		return ast.SkipParentheses(body)
	}

	statements := body.AsBlock().Statements.Nodes
	if len(statements) != 1 || !ast.IsReturnStatement(statements[0]) {
		return nil
	}

	ret := statements[0].AsReturnStatement()
	if ret == nil || ret.Expression == nil {
		return nil
	}

	return ast.SkipParentheses(ret.Expression)
}

func isParameterReference(node *ast.Node, parameterName string) bool {
	node = ast.SkipParentheses(node)
	return node != nil && ast.IsIdentifier(node) && node.AsIdentifier().Text == parameterName
}

func isNullishLiteral(node *ast.Node) bool {
	node = ast.SkipParentheses(node)
	if node == nil {
		return false
	}

	if node.Kind == ast.KindNullKeyword {
		return true
	}

	return ast.IsIdentifier(node) && node.AsIdentifier().Text == "undefined"
}

func isNullishComparison(node *ast.Node, parameterName string) bool {
	node = ast.SkipParentheses(node)
	if node == nil || !ast.IsBinaryExpression(node) {
		return false
	}

	binary := node.AsBinaryExpression()
	if binary == nil || binary.OperatorToken == nil {
		return false
	}

	switch binary.OperatorToken.Kind {
	case ast.KindExclamationEqualsToken, ast.KindExclamationEqualsEqualsToken:
		return (isParameterReference(binary.Left, parameterName) && isNullishLiteral(binary.Right)) ||
			(isNullishLiteral(binary.Left) && isParameterReference(binary.Right, parameterName))
	default:
		return false
	}
}

func isNullishGuardExpression(node *ast.Node, parameterName string) bool {
	node = ast.SkipParentheses(node)
	if node == nil {
		return false
	}

	if isNullishComparison(node, parameterName) {
		return true
	}

	if !ast.IsBinaryExpression(node) {
		return false
	}

	binary := node.AsBinaryExpression()
	if binary == nil || binary.OperatorToken == nil || binary.OperatorToken.Kind != ast.KindAmpersandAmpersandToken {
		return false
	}

	return isNullishGuardExpression(binary.Left, parameterName) && isNullishGuardExpression(binary.Right, parameterName)
}

func sameType(typeChecker *checker.Checker, left *checker.Type, right *checker.Type) bool {
	if left == nil || right == nil {
		return false
	}
	if utils.IsTypeAnyType(left) || utils.IsTypeAnyType(right) || utils.IsTypeUnknownType(left) || utils.IsTypeUnknownType(right) {
		return false
	}
	return checker.Checker_isTypeAssignableTo(typeChecker, left, right) && checker.Checker_isTypeAssignableTo(typeChecker, right, left)
}

func matchesNullablePredicate(typeChecker *checker.Checker, parameterType *checker.Type, predicateType *checker.Type) bool {
	if parameterType == nil || predicateType == nil {
		return false
	}

	parameterParts := utils.UnionTypeParts(parameterType)
	if len(parameterParts) <= 1 {
		return false
	}

	hasNullish := false
	nonNullishParts := make([]*checker.Type, 0, len(parameterParts))
	for _, part := range parameterParts {
		if part == nil {
			continue
		}
		if utils.IsTypeFlagSet(part, checker.TypeFlagsNull|checker.TypeFlagsUndefined) {
			hasNullish = true
			continue
		}
		nonNullishParts = append(nonNullishParts, part)
	}

	if !hasNullish {
		return false
	}

	predicateParts := utils.UnionTypeParts(predicateType)
	if len(nonNullishParts) != len(predicateParts) {
		return false
	}

	used := make([]bool, len(predicateParts))
	for _, parameterPart := range nonNullishParts {
		matched := false
		for i, predicatePart := range predicateParts {
			if used[i] || !sameType(typeChecker, parameterPart, predicatePart) {
				continue
			}
			used[i] = true
			matched = true
			break
		}
		if !matched {
			return false
		}
	}

	return true
}

func buildPreferIsTruthyMessage(parameterType string, predicateType string) rule.RuleMessage {
	return rule.RuleMessage{
		Id: "preferIsTruthy",
		Description: fmt.Sprintf(
			"Inline `filter` type guard from `%s` to `%s`. Use a reusable helper like `isTruthy` instead of an ad hoc nullable guard lambda.",
			parameterType,
			predicateType,
		),
	}
}

var PreferIsTruthyRule = rule.Rule{
	Name: "prefer-is-truthy",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		checkCallback := func(node *ast.Node) {
			if !isInlineFilterCallback(node) {
				return
			}

			predicate := getTypePredicate(node)
			if predicate == nil || predicate.ParameterName == nil || !ast.IsIdentifier(predicate.ParameterName) {
				return
			}

			parameter := findPredicateParameter(node, predicate)
			if parameter == nil || parameter.Name() == nil {
				return
			}

			parameterName := predicate.ParameterName.AsIdentifier().Text
			if !isNullishGuardExpression(getReturnedExpression(node), parameterName) {
				return
			}

			parameterType := ctx.TypeChecker.GetTypeAtLocation(parameter.Name())
			predicateType := checker.Checker_getTypeFromTypeNode(ctx.TypeChecker, predicate.Type)
			if !matchesNullablePredicate(ctx.TypeChecker, parameterType, predicateType) {
				return
			}

			ctx.ReportNode(node, buildPreferIsTruthyMessage(
				ctx.TypeChecker.TypeToString(parameterType),
				ctx.TypeChecker.TypeToString(predicateType),
			))
		}

		return rule.RuleListeners{
			ast.KindArrowFunction:      checkCallback,
			ast.KindFunctionExpression: checkCallback,
		}
	},
}
