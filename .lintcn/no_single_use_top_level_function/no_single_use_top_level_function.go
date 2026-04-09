// lintcn:name no-single-use-top-level-function
// lintcn:severity warn
// lintcn:description Warn on short top-level functions that are only called once in the program.

package no_single_use_top_level_function

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/lintcn-rules/program_refs"
)

const maxInlineableTopLevelFunctionLines = 12

func buildSingleUseTopLevelFunctionMessage(name string, lines int) rule.RuleMessage {
	return rule.RuleMessage{
		Id: "singleUseTopLevelFunction",
		Description: fmt.Sprintf(
			"Function `%s` is only called once in this program and is %d lines long. Inline it at the call site instead of creating extra cognitive overhead.",
			name,
			lines,
		),
	}
}

func isTopLevelFunctionDeclaration(node *ast.Node) bool {
	if node == nil || !ast.IsFunctionDeclaration(node) || node.Parent == nil {
		return false
	}
	if node.Parent.Kind != ast.KindSourceFile {
		return false
	}
	if node.Body() == nil || node.Name() == nil {
		return false
	}
	return !ast.HasSyntacticModifier(node, ast.ModifierFlagsAmbient|ast.ModifierFlagsDefault|ast.ModifierFlagsExport)
}

func isExportedViaNamedExports(typeChecker *checker.Checker, node *ast.Node, symbol *ast.Symbol) bool {
	if node == nil || symbol == nil {
		return false
	}

	sourceFile := ast.GetSourceFileOfNode(node)
	if sourceFile == nil {
		return false
	}

	for _, statement := range sourceFile.Statements.Nodes {
		if !ast.IsExportDeclaration(statement) {
			continue
		}

		exportDecl := statement.AsExportDeclaration()
		if exportDecl.IsTypeOnly || exportDecl.ModuleSpecifier != nil || exportDecl.ExportClause == nil || !ast.IsNamedExports(exportDecl.ExportClause.AsNode()) {
			continue
		}

		for _, specifierNode := range exportDecl.ExportClause.AsNamedExports().Elements.Nodes {
			specifier := specifierNode.AsExportSpecifier()
			if specifier.IsTypeOnly {
				continue
			}

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

func declarationLineCount(node *ast.Node) int {
	if node == nil {
		return 0
	}
	sourceFile := ast.GetSourceFileOfNode(node)
	if sourceFile == nil {
		return 0
	}
	startLine, _ := scanner.GetECMALineAndUTF16CharacterOfPosition(sourceFile, node.Pos())
	endLine, _ := scanner.GetECMALineAndUTF16CharacterOfPosition(sourceFile, node.End())
	return endLine - startLine + 1
}

var NoSingleUseTopLevelFunctionRule = rule.Rule{
	Name: "no-single-use-top-level-function",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindFunctionDeclaration: func(node *ast.Node) {
				if !isTopLevelFunctionDeclaration(node) {
					return
				}

				lineCount := declarationLineCount(node)
				if lineCount == 0 || lineCount >= maxInlineableTopLevelFunctionLines {
					return
				}

				name := node.Name()
				symbol := program_refs.SymbolAtLocation(ctx.TypeChecker, name)
				if symbol == nil {
					return
				}
				if isExportedViaNamedExports(ctx.TypeChecker, node, symbol) {
					return
				}

				searchOptions := program_refs.FindOptions{ExcludeWithin: node}
				callReferences := program_refs.FindCallReferences(ctx.Program, ctx.TypeChecker, symbol, searchOptions)
				if len(callReferences) != 1 {
					return
				}

				allReferences := program_refs.FindSymbolReferences(ctx.Program, ctx.TypeChecker, symbol, searchOptions)
				if len(allReferences) != 1 {
					return
				}

				ctx.ReportNode(name, buildSingleUseTopLevelFunctionMessage(name.Text(), lineCount))
			},
		}
	},
}
