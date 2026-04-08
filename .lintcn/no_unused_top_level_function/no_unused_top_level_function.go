// lintcn:name no-unused-top-level-function
// lintcn:severity warn
// lintcn:description Warn on top-level functions that have no callers or other references in the program.

package no_unused_top_level_function

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/lintcn-rules/program_refs"
)

func buildUnusedTopLevelFunctionMessage(name string, callers int) rule.RuleMessage {
	return rule.RuleMessage{
		Id: "unusedTopLevelFunction",
		Description: fmt.Sprintf(
			"Top-level function `%s` has %d callers in this program. Inline it at the use site, delete it, or split the logic into a clearer abstraction.",
			name,
			callers,
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

var NoUnusedTopLevelFunctionRule = rule.Rule{
	Name: "no-unused-top-level-function",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindFunctionDeclaration: func(node *ast.Node) {
				if !isTopLevelFunctionDeclaration(node) {
					return
				}

				name := node.Name()
				symbol := program_refs.SymbolAtLocation(ctx.TypeChecker, name)
				if symbol == nil {
					return
				}

				searchOptions := program_refs.FindOptions{ExcludeWithin: node}
				callReferences := program_refs.FindCallReferences(ctx.Program, ctx.TypeChecker, symbol, searchOptions)
				if len(callReferences) != 0 {
					return
				}

				allReferences := program_refs.FindSymbolReferences(ctx.Program, ctx.TypeChecker, symbol, searchOptions)
				if len(allReferences) != 0 {
					return
				}

				ctx.ReportNode(name, buildUnusedTopLevelFunctionMessage(name.Text(), len(callReferences)))
			},
		}
	},
}
