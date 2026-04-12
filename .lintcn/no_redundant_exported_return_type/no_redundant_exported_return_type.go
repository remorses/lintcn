// lintcn:name no-redundant-exported-return-type
// lintcn:description Warn on exported ReturnType aliases when the resolved named type can be referenced directly.
//
// Package no_redundant_exported_return_type detects exported type aliases that
// hide an already-nameable return type behind ReturnType<...>.
package no_redundant_exported_return_type

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
	"github.com/typescript-eslint/tsgolint/lintcn-rules/program_refs"
)

func buildRedundantExportedReturnTypeMessage(original string, direct string) rule.RuleMessage {
	return rule.RuleMessage{
		Id: "redundantExportedReturnType",
		Description: fmt.Sprintf(
			"`%s` hides an exported return type. Use `%s` directly instead.",
			original,
			direct,
		),
	}
}

func getNodeText(sourceFile *ast.SourceFile, node *ast.Node) string {
	if sourceFile == nil || node == nil {
		return ""
	}
	textRange := utils.TrimNodeTextRange(sourceFile, node)
	if textRange.Pos() < 0 || textRange.End() > len(sourceFile.Text()) || textRange.Pos() >= textRange.End() {
		return ""
	}
	return sourceFile.Text()[textRange.Pos():textRange.End()]
}

func isReturnTypeReference(node *ast.Node) bool {
	if node == nil || !ast.IsTypeReferenceNode(node) {
		return false
	}
	ref := node.AsTypeReferenceNode()
	if ref.TypeName == nil || ref.TypeArguments == nil || len(ref.TypeArguments.Nodes) != 1 {
		return false
	}
	return ast.IsIdentifier(ref.TypeName.AsNode()) && ref.TypeName.AsNode().AsIdentifier().Text == "ReturnType"
}

func resolveReturnTypeSource(ctx rule.RuleContext, node *ast.Node) *ast.Node {
	if node == nil {
		return nil
	}

	switch {
	case ast.IsTypeQueryNode(node):
		typeQuery := node.AsTypeQueryNode()
		if typeQuery.ExprName == nil {
			return nil
		}
		symbol := program_refs.SymbolAtLocation(ctx.TypeChecker, typeQuery.ExprName.AsNode())
		if symbol == nil {
			return nil
		}
		for _, declaration := range symbol.Declarations {
			if returnType := returnTypeFromDeclaration(declaration); returnType != nil {
				return returnType
			}
		}
	case ast.IsTypeReferenceNode(node):
		ref := node.AsTypeReferenceNode()
		if ref.TypeName == nil {
			return nil
		}
		symbol := program_refs.SymbolAtLocation(ctx.TypeChecker, ref.TypeName.AsNode())
		if symbol == nil {
			return nil
		}
		for _, declaration := range symbol.Declarations {
			if !ast.IsTypeAliasDeclaration(declaration) {
				continue
			}
			aliasDecl := declaration.AsTypeAliasDeclaration()
			if aliasDecl.Type != nil && ast.IsFunctionTypeNode(aliasDecl.Type.AsNode()) {
				return aliasDecl.Type.AsNode().AsFunctionTypeNode().Type.AsNode()
			}
		}
	}

	return nil
}

func returnTypeArgumentSymbol(ctx rule.RuleContext, node *ast.Node) *ast.Symbol {
	if node == nil {
		return nil
	}

	switch {
	case ast.IsTypeQueryNode(node):
		typeQuery := node.AsTypeQueryNode()
		if typeQuery.ExprName == nil {
			return nil
		}
		return program_refs.SymbolAtLocation(ctx.TypeChecker, typeQuery.ExprName.AsNode())
	case ast.IsTypeReferenceNode(node):
		ref := node.AsTypeReferenceNode()
		if ref.TypeName == nil {
			return nil
		}
		return program_refs.SymbolAtLocation(ctx.TypeChecker, ref.TypeName.AsNode())
	default:
		return nil
	}
}

func resolveAliasedSymbol(typeChecker *checker.Checker, symbol *ast.Symbol) *ast.Symbol {
	if typeChecker == nil || symbol == nil {
		return symbol
	}
	if symbol.Flags&ast.SymbolFlagsAlias != 0 {
		if aliased := checker.Checker_GetAliasedSymbol(typeChecker, symbol); aliased != nil {
			return aliased
		}
	}
	return symbol
}

func getEnclosingDeclaration(node *ast.Node) *ast.Node {
	for current := node; current != nil; current = current.Parent {
		if ast.IsDeclaration(current) || ast.IsSourceFile(current) {
			return current
		}
	}
	return nil
}

func returnTypeFromDeclaration(node *ast.Node) *ast.Node {
	if node == nil {
		return nil
	}

	switch {
	case ast.IsFunctionDeclaration(node), ast.IsFunctionExpression(node), ast.IsArrowFunction(node), ast.IsMethodDeclaration(node):
		return node.Type()
	case ast.IsVariableDeclaration(node):
		decl := node.AsVariableDeclaration()
		if decl.Type != nil && ast.IsFunctionTypeNode(decl.Type.AsNode()) {
			return decl.Type.AsNode().AsFunctionTypeNode().Type.AsNode()
		}
		if decl.Initializer != nil {
			initializer := decl.Initializer.AsNode()
			if ast.IsFunctionExpression(initializer) || ast.IsArrowFunction(initializer) {
				return initializer.Type()
			}
		}
	}

	return nil
}

func isExportedDeclaration(typeChecker *checker.Checker, declaration *ast.Node, symbol *ast.Symbol) bool {
	if typeChecker == nil || declaration == nil || symbol == nil {
		return false
	}

	if ast.HasSyntacticModifier(declaration, ast.ModifierFlagsExport|ast.ModifierFlagsDefault) {
		return true
	}

	sourceFile := ast.GetSourceFileOfNode(declaration)
	if sourceFile == nil {
		return false
	}

	for _, statement := range sourceFile.Statements.Nodes {
		if !ast.IsExportDeclaration(statement) {
			continue
		}
		exportDecl := statement.AsExportDeclaration()
		if exportDecl.ModuleSpecifier != nil || exportDecl.ExportClause == nil || !ast.IsNamedExports(exportDecl.ExportClause.AsNode()) {
			continue
		}
		for _, specifierNode := range exportDecl.ExportClause.AsNamedExports().Elements.Nodes {
			specifier := specifierNode.AsExportSpecifier()
			localName := specifier.Name().AsNode()
			if specifier.PropertyName != nil {
				localName = specifier.PropertyName.AsNode()
			}
			if program_refs.SymbolAtLocation(typeChecker, localName) == symbol {
				return true
			}
		}
	}

	return false
}

func isExportedSymbol(typeChecker *checker.Checker, symbol *ast.Symbol) bool {
	if typeChecker == nil || symbol == nil {
		return false
	}
	symbol = resolveAliasedSymbol(typeChecker, symbol)
	for _, declaration := range symbol.Declarations {
		if declaration == nil {
			continue
		}
		if isExportedDeclaration(typeChecker, declaration, symbol) {
			return true
		}
	}
	return false
}

func isTypeParameterSymbol(symbol *ast.Symbol) bool {
	if symbol == nil {
		return false
	}
	for _, declaration := range symbol.Declarations {
		if declaration != nil && ast.IsTypeParameterDeclaration(declaration) {
			return true
		}
	}
	return false
}

func isUsefulDirectTypeText(text string) bool {
	if text == "" {
		return false
	}
	if strings.Contains(text, "ReturnType<") || strings.Contains(text, "{") || strings.Contains(text, "=>") {
		return false
	}
	switch text {
	case "string", "number", "boolean", "bigint", "symbol", "null", "undefined", "void", "unknown", "any", "never":
		return false
	default:
		return true
	}
}

func isAccessibleTypeNode(ctx rule.RuleContext, enclosingDeclaration *ast.Node, node *ast.Node) bool {
	if node == nil {
		return false
	}

	switch node.Kind {
	case ast.KindStringKeyword,
		ast.KindNumberKeyword,
		ast.KindBooleanKeyword,
		ast.KindBigIntKeyword,
		ast.KindSymbolKeyword,
		ast.KindNullKeyword,
		ast.KindUndefinedKeyword,
		ast.KindVoidKeyword,
		ast.KindUnknownKeyword,
		ast.KindAnyKeyword,
		ast.KindNeverKeyword,
		ast.KindObjectKeyword,
		ast.KindThisType:
		return true
	case ast.KindTypeReference:
		ref := node.AsTypeReferenceNode()
		if ref.TypeName == nil {
			return false
		}
		symbol := resolveAliasedSymbol(ctx.TypeChecker, program_refs.SymbolAtLocation(ctx.TypeChecker, ref.TypeName.AsNode()))
		if symbol == nil {
			return false
		}
		if !isTypeParameterSymbol(symbol) && !utils.IsSymbolFromDefaultLibrary(ctx.Program, symbol) && !isExportedSymbol(ctx.TypeChecker, symbol) {
			return false
		}
		if !isTypeParameterSymbol(symbol) && !checker.Checker_IsTypeSymbolAccessible(ctx.TypeChecker, symbol, enclosingDeclaration) {
			return false
		}
		if ref.TypeArguments == nil {
			return true
		}
		for _, typeArgument := range ref.TypeArguments.Nodes {
			if !isAccessibleTypeNode(ctx, enclosingDeclaration, typeArgument) {
				return false
			}
		}
		return true
	case ast.KindUnionType:
		for _, part := range node.AsUnionTypeNode().Types.Nodes {
			if !isAccessibleTypeNode(ctx, enclosingDeclaration, part) {
				return false
			}
		}
		return true
	case ast.KindIntersectionType:
		for _, part := range node.AsIntersectionTypeNode().Types.Nodes {
			if !isAccessibleTypeNode(ctx, enclosingDeclaration, part) {
				return false
			}
		}
		return true
	case ast.KindArrayType:
		return isAccessibleTypeNode(ctx, enclosingDeclaration, node.AsArrayTypeNode().ElementType.AsNode())
	case ast.KindTupleType:
		for _, element := range node.AsTupleTypeNode().Elements.Nodes {
			if !isAccessibleTypeNode(ctx, enclosingDeclaration, element) {
				return false
			}
		}
		return true
	case ast.KindParenthesizedType, ast.KindTypeOperator:
		return isAccessibleTypeNode(ctx, enclosingDeclaration, node.Type())
	case ast.KindTypeQuery:
		typeQuery := node.AsTypeQueryNode()
		if typeQuery.ExprName == nil {
			return false
		}
		symbol := resolveAliasedSymbol(ctx.TypeChecker, program_refs.SymbolAtLocation(ctx.TypeChecker, typeQuery.ExprName.AsNode()))
		if symbol == nil {
			return false
		}
		if !isTypeParameterSymbol(symbol) && !utils.IsSymbolFromDefaultLibrary(ctx.Program, symbol) && !isExportedSymbol(ctx.TypeChecker, symbol) {
			return false
		}
		return isTypeParameterSymbol(symbol) || checker.Checker_IsValueSymbolAccessible(ctx.TypeChecker, symbol, enclosingDeclaration)
	case ast.KindLiteralType:
		return true
	default:
		return false
	}
}

func isAccessibleType(ctx rule.RuleContext, enclosingDeclaration *ast.Node, t *checker.Type, seen map[*checker.Type]bool) bool {
	if t == nil {
		return false
	}
	if seen[t] {
		return true
	}
	seen[t] = true

	if alias := checker.Type_alias(t); alias != nil {
		symbol := resolveAliasedSymbol(ctx.TypeChecker, alias.Symbol())
		if symbol == nil {
			return false
		}
		if !isTypeParameterSymbol(symbol) && !utils.IsSymbolFromDefaultLibrary(ctx.Program, symbol) && !isExportedSymbol(ctx.TypeChecker, symbol) {
			return false
		}
		if !isTypeParameterSymbol(symbol) && !checker.Checker_IsTypeSymbolAccessible(ctx.TypeChecker, symbol, enclosingDeclaration) {
			return false
		}
		for _, typeArgument := range alias.TypeArguments() {
			if !isAccessibleType(ctx, enclosingDeclaration, typeArgument, seen) {
				return false
			}
		}
		return true
	}

	unionParts := utils.UnionTypeParts(t)
	if len(unionParts) > 1 {
		for _, part := range unionParts {
			if !isAccessibleType(ctx, enclosingDeclaration, part, seen) {
				return false
			}
		}
		return true
	}

	intersectionParts := utils.IntersectionTypeParts(t)
	if len(intersectionParts) > 1 {
		for _, part := range intersectionParts {
			if !isAccessibleType(ctx, enclosingDeclaration, part, seen) {
				return false
			}
		}
		return true
	}

	if symbol := resolveAliasedSymbol(ctx.TypeChecker, checker.Type_symbol(t)); symbol != nil {
		if !isTypeParameterSymbol(symbol) && !utils.IsSymbolFromDefaultLibrary(ctx.Program, symbol) && !isExportedSymbol(ctx.TypeChecker, symbol) {
			return false
		}
		if !isTypeParameterSymbol(symbol) && !checker.Checker_IsTypeSymbolAccessible(ctx.TypeChecker, symbol, enclosingDeclaration) {
			return false
		}
	}

	typeArguments := []*checker.Type(nil)
	if checker.Type_flags(t)&checker.TypeFlagsObject != 0 && checker.Type_objectFlags(t)&checker.ObjectFlagsReference != 0 {
		typeArguments = checker.Checker_getTypeArguments(ctx.TypeChecker, t)
	}
	for _, typeArgument := range typeArguments {
		if !isAccessibleType(ctx, enclosingDeclaration, typeArgument, seen) {
			return false
		}
	}

	if checker.Type_flags(t)&checker.TypeFlagsObject != 0 && checker.Type_symbol(t) == nil && len(typeArguments) == 0 {
		return false
	}

	return true
}

var NoRedundantExportedReturnTypeRule = rule.Rule{
	Name: "no-redundant-exported-return-type",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		checkReturnTypeReference := func(node *ast.Node) {
			if node == nil || !isReturnTypeReference(node) {
				return
			}

			returnTypeRef := node.AsTypeReferenceNode()
			returnTypeArg := returnTypeRef.TypeArguments.Nodes[0]
			targetSymbol := returnTypeArgumentSymbol(ctx, returnTypeArg)
			returnTypeSource := resolveReturnTypeSource(ctx, returnTypeArg)
			enclosingDeclaration := getEnclosingDeclaration(node)
			if targetSymbol == nil || !isExportedSymbol(ctx.TypeChecker, targetSymbol) || returnTypeSource == nil {
				return
			}
			if enclosingDeclaration == nil || !isAccessibleTypeNode(ctx, enclosingDeclaration, returnTypeSource) {
				return
			}

			resolvedType := checker.Checker_getTypeFromTypeNode(ctx.TypeChecker, node)
			if resolvedType == nil || !isAccessibleType(ctx, enclosingDeclaration, resolvedType, map[*checker.Type]bool{}) {
				return
			}

			directTypeText := ctx.TypeChecker.TypeToString(resolvedType)
			if !isUsefulDirectTypeText(directTypeText) {
				return
			}

			originalText := getNodeText(ctx.SourceFile, node)
			if originalText == "" || originalText == directTypeText {
				return
			}

			ctx.ReportNode(node, buildRedundantExportedReturnTypeMessage(originalText, directTypeText))
		}

		return rule.RuleListeners{
			ast.KindTypeReference: checkReturnTypeReference,
		}
	},
}
