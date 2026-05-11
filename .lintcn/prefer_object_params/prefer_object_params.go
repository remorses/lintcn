// lintcn:name prefer-object-params
// lintcn:severity warn
// lintcn:description Warn on reusable function definitions with 3 or more positional parameters and prefer a single object parameter.
//
// Package prefer_object_params warns when reusable function definitions grow to
// three or more positional parameters.
package prefer_object_params

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func buildPreferObjectParamsMessage(kind string, name string, count int) rule.RuleMessage {
	description := fmt.Sprintf(
		"This %s has %d positional parameters. Use one object parameter instead.",
		kind,
		count,
	)
	if name != "" {
		subject := fmt.Sprintf("%s `%s`", kind, name)
		if kind == "constructor" {
			subject = fmt.Sprintf("constructor for `%s`", name)
		}
		description = fmt.Sprintf(
			"%s has %d positional parameters. Use one object parameter instead.",
			subject,
			count,
		)
	}
	return rule.RuleMessage{
		Id:          "preferObjectParams",
		Description: description,
	}
}

func countPositionalParameters(parameters []*ast.ParameterDeclarationNode) int {
	count := 0
	for _, parameter := range parameters {
		if parameter == nil || ast.IsThisParameter(parameter.AsNode()) {
			continue
		}
		count++
	}
	return count
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

func shouldCheckMethodDeclaration(node *ast.Node) bool {
	if node == nil || node.Parent == nil {
		return false
	}
	if node.Parent.Kind == ast.KindObjectLiteralExpression {
		return isNamedObjectLiteralContext(node.Parent)
	}
	return true
}

func definitionName(node *ast.Node) string {
	if node == nil {
		return ""
	}
	if name := node.Name(); name != nil {
		return supportedNameText(name)
	}
	if node.Parent != nil && node.Parent.Name() != nil {
		return supportedNameText(node.Parent.Name())
	}
	return ""
}

func supportedNameText(node *ast.Node) string {
	if node == nil {
		return ""
	}
	switch node.Kind {
	case ast.KindIdentifier,
		ast.KindPrivateIdentifier,
		ast.KindStringLiteral,
		ast.KindNumericLiteral,
		ast.KindBigIntLiteral,
		ast.KindNoSubstitutionTemplateLiteral:
		return node.Text()
	default:
		return ""
	}
}

func messageNodeForDefinition(node *ast.Node) *ast.Node {
	if node == nil {
		return nil
	}
	if name := node.Name(); name != nil {
		return name
	}
	if node.Parent != nil && node.Parent.Name() != nil {
		return node.Parent.Name()
	}
	return node
}

var PreferObjectParamsRule = rule.Rule{
	Name: "prefer-object-params",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		checkFunctionLike := func(node *ast.Node, kind string) {
			if node == nil || node.Body() == nil {
				return
			}

			parameterCount := countPositionalParameters(node.Parameters())
			if parameterCount < 3 {
				return
			}

			messageNode := messageNodeForDefinition(node)
			if messageNode == nil {
				messageNode = node
			}

			ctx.ReportNode(messageNode, buildPreferObjectParamsMessage(kind, definitionName(node), parameterCount))
		}

		return rule.RuleListeners{
			ast.KindFunctionDeclaration: func(node *ast.Node) {
				checkFunctionLike(node, "function")
			},
			ast.KindFunctionExpression: func(node *ast.Node) {
				if !shouldCheckFunctionExpressionLike(node) {
					return
				}
				checkFunctionLike(node, "function")
			},
			ast.KindArrowFunction: func(node *ast.Node) {
				if !shouldCheckFunctionExpressionLike(node) {
					return
				}
				checkFunctionLike(node, "function")
			},
			ast.KindMethodDeclaration: func(node *ast.Node) {
				if !shouldCheckMethodDeclaration(node) {
					return
				}
				checkFunctionLike(node, "method")
			},
			ast.KindConstructor: func(node *ast.Node) {
				checkFunctionLike(node, "constructor")
			},
		}
	},
}
