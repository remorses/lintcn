// lintcn:name no-tiny-wrapper-function
// lintcn:severity warn
// lintcn:description Warn on functions and methods that only forward directly to another call.

// Package no_tiny_wrapper_function warns on direct pass-through wrapper
// functions and methods so agents inline direct calls instead of adding
// throwaway abstractions.
package no_tiny_wrapper_function

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildTinyWrapperMessage(kind string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "tinyWrapper",
		Description: "This " + kind + " only forwards directly to another call. Inline the call instead of creating a wrapper function.",
	}
}

func getWrappedExpression(node *ast.Node) (*ast.Node, bool) {
	if node == nil || node.Body() == nil {
		return nil, false
	}

	body := ast.SkipParentheses(node.Body())
	if body == nil {
		return nil, false
	}

	if !ast.IsBlock(body) {
		return unwrapAwaitExpression(body), true
	}

	statements := body.AsBlock().Statements.Nodes
	if len(statements) != 1 {
		return nil, false
	}

	statement := statements[0]
	switch statement.Kind {
	case ast.KindReturnStatement:
		ret := statement.AsReturnStatement()
		if ret.Expression == nil {
			return nil, false
		}
		return unwrapAwaitExpression(ret.Expression), true
	case ast.KindExpressionStatement:
		return unwrapAwaitExpression(statement.AsExpressionStatement().Expression), true
	default:
		return nil, false
	}
}

func unwrapAwaitExpression(node *ast.Node) *ast.Node {
	current := ast.SkipParentheses(node)
	for current != nil && ast.IsAwaitExpression(current) {
		current = ast.SkipParentheses(current.AsAwaitExpression().Expression)
	}
	return current
}

func isSimpleCallee(node *ast.Node) bool {
	node = ast.SkipParentheses(node)
	if node == nil {
		return false
	}

	switch {
	case ast.IsIdentifier(node):
		return true
	case ast.IsPropertyAccessExpression(node):
		return isSimpleCallee(node.AsPropertyAccessExpression().Expression)
	case ast.IsElementAccessExpression(node):
		return isSimpleCallee(node.AsElementAccessExpression().Expression) && isSimpleForwardedArgument(node.AsElementAccessExpression().ArgumentExpression)
	default:
		return node.Kind == ast.KindThisKeyword || node.Kind == ast.KindSuperKeyword
	}
}

func isSimpleForwardedArgument(node *ast.Node) bool {
	node = ast.SkipParentheses(node)
	if node == nil {
		return false
	}

	if ast.IsSpreadElement(node) {
		return isSimpleForwardedArgument(node.AsSpreadElement().Expression)
	}

	// Treat computed arguments as meaningful behavior so small adapter functions
	// like `write(normalize(value))` or `write(user.name)` stay allowed.
	return ast.IsIdentifier(node) || node.Kind == ast.KindThisKeyword || node.Kind == ast.KindSuperKeyword
}

func allArgumentsAreSimple(arguments *ast.NodeList) bool {
	if arguments == nil {
		return true
	}

	for _, arg := range arguments.Nodes {
		if !isSimpleForwardedArgument(arg) {
			return false
		}
	}

	return true
}

func forwardsDirectlyToAnotherCall(node *ast.Node) bool {
	node = ast.SkipParentheses(node)
	if node == nil {
		return false
	}

	switch node.Kind {
	case ast.KindCallExpression:
		call := node.AsCallExpression()
		if !isSimpleCallee(call.Expression) {
			return false
		}
		return allArgumentsAreSimple(call.Arguments)
	case ast.KindNewExpression:
		newExpr := node.AsNewExpression()
		if !isSimpleCallee(newExpr.Expression) {
			return false
		}
		return allArgumentsAreSimple(newExpr.Arguments)
	default:
		return false
	}
}

func isNamedObjectLiteralContext(node *ast.Node) bool {
	if node == nil || node.Parent == nil {
		return false
	}

	parent := node.Parent
	switch {
	case ast.IsVariableDeclaration(parent):
		return parent.Initializer() == node
	case ast.IsPropertyDeclaration(parent):
		return parent.Initializer() == node
	case ast.IsExportAssignment(parent):
		return parent.Expression() == node
	case ast.IsBinaryExpression(parent):
		bin := parent.AsBinaryExpression()
		return bin != nil && bin.OperatorToken != nil && bin.OperatorToken.Kind == ast.KindEqualsToken && bin.Right == node && ast.IsAssignmentTarget(bin.Left)
	default:
		return false
	}
}

func shouldCheckFunctionExpressionLike(node *ast.Node) bool {
	if node == nil || node.Parent == nil {
		return false
	}

	parent := node.Parent
	switch {
	case ast.IsVariableDeclaration(parent):
		return parent.Initializer() == node
	case ast.IsPropertyDeclaration(parent):
		return parent.Initializer() == node
	case ast.IsExportAssignment(parent):
		return parent.Expression() == node
	case ast.IsBinaryExpression(parent):
		bin := parent.AsBinaryExpression()
		return bin != nil && bin.OperatorToken != nil && bin.OperatorToken.Kind == ast.KindEqualsToken && bin.Right == node && ast.IsAssignmentTarget(bin.Left)
	case ast.IsPropertyAssignment(parent):
		property := parent.AsPropertyAssignment()
		if property == nil || property.Initializer != node || parent.Parent == nil || parent.Parent.Kind != ast.KindObjectLiteralExpression {
			return false
		}
		return isNamedObjectLiteralContext(parent.Parent)
	default:
		return false
	}
}

func checkTinyWrapper(ctx rule.RuleContext, node *ast.Node, kind string) {
	wrappedExpr, ok := getWrappedExpression(node)
	if !ok || !forwardsDirectlyToAnotherCall(wrappedExpr) {
		return
	}

	messageNode := node
	if name := node.Name(); name != nil {
		messageNode = name
	}

	// Report on the declaration name when available so the warning points at the
	// abstraction being introduced, not only its body contents.
	ctx.ReportNode(messageNode, buildTinyWrapperMessage(kind))
}

var NoTinyWrapperFunctionRule = rule.Rule{
	Name: "no-tiny-wrapper-function",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindFunctionDeclaration: func(node *ast.Node) {
				checkTinyWrapper(ctx, node, "function")
			},
			ast.KindFunctionExpression: func(node *ast.Node) {
				if !shouldCheckFunctionExpressionLike(node) {
					return
				}
				checkTinyWrapper(ctx, node, "function")
			},
			ast.KindArrowFunction: func(node *ast.Node) {
				if !shouldCheckFunctionExpressionLike(node) {
					return
				}
				checkTinyWrapper(ctx, node, "function")
			},
			ast.KindMethodDeclaration: func(node *ast.Node) {
				checkTinyWrapper(ctx, node, "method")
			},
		}
	},
}
