// lintcn:name no-redundant-contextual-parameter-type
// lintcn:description Disallow explicit callback parameter type annotations when the same type is already provided contextually.
//
// Package no_redundant_contextual_parameter_type detects redundant parameter
// annotations on contextually typed callbacks.
package no_redundant_contextual_parameter_type

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func parameterIndex(parameters []*ast.ParameterDeclarationNode, target *ast.Node) int {
	for i, parameter := range parameters {
		if parameter != nil && parameter.AsNode() == target {
			return i
		}
	}
	return -1
}

func containsTypeParameter(t *checker.Type) bool {
	if t == nil {
		return false
	}
	if utils.IsTypeParameter(t) {
		return true
	}
	for _, part := range utils.UnionTypeParts(t) {
		if part != nil && part != t && containsTypeParameter(part) {
			return true
		}
	}
	for _, part := range utils.IntersectionTypeParts(t) {
		if part != nil && part != t && containsTypeParameter(part) {
			return true
		}
	}
	return false
}

func isRedundantContextualType(typeChecker *checker.Checker, declaredType *checker.Type, contextualType *checker.Type) bool {
	if declaredType == nil || contextualType == nil {
		return false
	}
	if utils.IsTypeAnyType(declaredType) || utils.IsTypeAnyType(contextualType) {
		return false
	}
	if utils.IsTypeUnknownType(declaredType) || utils.IsTypeUnknownType(contextualType) {
		return false
	}
	if containsTypeParameter(declaredType) || containsTypeParameter(contextualType) {
		return false
	}
	return checker.Checker_isTypeAssignableTo(typeChecker, declaredType, contextualType) && checker.Checker_isTypeAssignableTo(typeChecker, contextualType, declaredType)
}

func isGenericInferenceCallback(functionNode *ast.Node, typeChecker *checker.Checker) bool {
	if functionNode == nil || functionNode.Parent == nil {
		return false
	}

	parent := functionNode.Parent
	if !ast.IsCallExpression(parent) && !ast.IsNewExpression(parent) {
		return false
	}

	resolvedSignature := typeChecker.GetResolvedSignature(parent)
	if resolvedSignature == nil {
		return false
	}
	if len(resolvedSignature.TypeParameters()) > 0 {
		return true
	}
	declaration := checker.Signature_declaration(resolvedSignature)
	return declaration != nil && len(declaration.TypeParameters()) > 0
}

var NoRedundantContextualParameterTypeRule = rule.Rule{
	Name: "no-redundant-contextual-parameter-type",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		checkParameter := func(node *ast.Node) {
			if node == nil || ast.IsThisParameter(node) {
				return
			}

			param := node.AsParameterDeclaration()
			if param == nil || param.Type == nil || param.DotDotDotToken != nil {
				return
			}

			parent := node.Parent
			if parent == nil || (!ast.IsArrowFunction(parent) && !ast.IsFunctionExpression(parent)) {
				return
			}
			if len(parent.TypeParameters()) > 0 {
				return
			}
			if isGenericInferenceCallback(parent, ctx.TypeChecker) {
				return
			}

			paramIndex := parameterIndex(parent.Parameters(), node)
			if paramIndex == -1 {
				return
			}

			contextualType := checker.Checker_getContextualType(ctx.TypeChecker, parent, checker.ContextFlagsNone)
			if contextualType == nil {
				return
			}

			signatures := utils.GetCallSignatures(ctx.TypeChecker, contextualType)
			if len(signatures) != 1 || checker.Signature_declaration(signatures[0]) == parent {
				return
			}

			signature := signatures[0]
			if signature.ThisParameter() != nil && len(parent.Parameters()) > 0 {
				firstParameter := parent.Parameters()[0].AsNode()
				if firstParameter != nil && firstParameter.Name() != nil && ast.IsIdentifier(firstParameter.Name()) && firstParameter.Name().AsIdentifier().Text == "this" {
					paramIndex--
				}
			}
			if paramIndex < 0 {
				return
			}

			contextualParameters := checker.Signature_parameters(signature)
			if paramIndex >= len(contextualParameters) {
				return
			}

			contextualParam := contextualParameters[paramIndex]
			if contextualParam == nil {
				return
			}
			if contextualParam.ValueDeclaration != nil && contextualParam.ValueDeclaration.Kind == ast.KindParameter && contextualParam.ValueDeclaration.AsParameterDeclaration().DotDotDotToken != nil {
				return
			}

			declaredType := checker.Checker_getTypeFromTypeNode(ctx.TypeChecker, param.Type)
			contextualParamType := checker.Checker_getTypeOfSymbol(ctx.TypeChecker, contextualParam)
			if !isRedundantContextualType(ctx.TypeChecker, declaredType, contextualParamType) {
				return
			}

			ctx.ReportNode(param.Type, rule.RuleMessage{
				Id: "redundantContextualParameterType",
				Description: fmt.Sprintf(
					"Parameter type annotation is redundant because the callback context already provides `%s`. Remove the explicit annotation.",
					ctx.TypeChecker.TypeToString(contextualParamType),
				),
			})
		}

		return rule.RuleListeners{
			ast.KindParameter: checkParameter,
		}
	},
}
