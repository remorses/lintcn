// lintcn:name no-tiny-wrapper-function
// lintcn:severity warn
// lintcn:description Warn on functions and methods with three or fewer lines of code that only exist to call other functions.

// Package no_tiny_wrapper_function warns on tiny wrapper functions and methods
// so agents inline direct calls instead of adding throwaway abstractions.
package no_tiny_wrapper_function

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

const maxWrapperLines = 3

func countCodeLines(text string, start int, end int) int {
	if start < 0 {
		start = 0
	}
	if end < start || start >= len(text) {
		return 0
	}
	if end > len(text) {
		end = len(text)
	}

	count := 0
	for _, line := range strings.Split(text[start:end], "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func executableRange(node *ast.Node) (int, int, bool) {
	if node == nil || node.Body() == nil {
		return 0, 0, false
	}

	body := ast.SkipParentheses(node.Body())
	if body == nil {
		return 0, 0, false
	}

	if ast.IsBlock(body) {
		statements := body.AsBlock().Statements.Nodes
		if len(statements) == 0 {
			return 0, 0, false
		}
		return statements[0].Pos(), statements[len(statements)-1].End(), true
	}

	return body.Pos(), body.End(), true
}

func containsCallExpression(node *ast.Node, root *ast.Node) bool {
	if node == nil {
		return false
	}
	if node != root && ast.IsFunctionLike(node) {
		return false
	}
	if node.Kind == ast.KindCallExpression {
		return true
	}
	return node.ForEachChild(func(child *ast.Node) bool {
		return containsCallExpression(child, root)
	})
}

func buildTinyWrapperMessage(kind string, lines int) rule.RuleMessage {
	lineWord := "lines"
	if lines == 1 {
		lineWord = "line"
	}
	return rule.RuleMessage{
		Id: "tinyWrapper",
		Description: fmt.Sprintf(
			"This %s is only %d %s of code and contains a function call. Inline the call directly instead of creating a wrapper function.",
			kind,
			lines,
			lineWord,
		),
	}
}

func checkTinyWrapper(ctx rule.RuleContext, node *ast.Node, kind string) {
	start, end, ok := executableRange(node)
	if !ok {
		return
	}

	body := ast.SkipParentheses(node.Body())
	if body == nil || !containsCallExpression(body, body) {
		return
	}

	lines := countCodeLines(ctx.SourceFile.Text(), start, end)
	if lines == 0 || lines > maxWrapperLines {
		return
	}

	messageNode := node
	if name := node.Name(); name != nil {
		messageNode = name
	}

	// Report on the declaration name when available so the warning points at the
	// abstraction being introduced, not only its body contents.
	ctx.ReportNode(messageNode, buildTinyWrapperMessage(kind, lines))
}

var NoTinyWrapperFunctionRule = rule.Rule{
	Name: "no-tiny-wrapper-function",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindFunctionDeclaration: func(node *ast.Node) {
				checkTinyWrapper(ctx, node, "function")
			},
			ast.KindFunctionExpression: func(node *ast.Node) {
				checkTinyWrapper(ctx, node, "function")
			},
			ast.KindArrowFunction: func(node *ast.Node) {
				checkTinyWrapper(ctx, node, "function")
			},
			ast.KindMethodDeclaration: func(node *ast.Node) {
				checkTinyWrapper(ctx, node, "method")
			},
		}
	},
}
