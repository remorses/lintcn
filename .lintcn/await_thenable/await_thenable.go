// lintcn:source https://github.com/oxc-project/tsgolint/tree/main/internal/rules/await_thenable
package await_thenable

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// anyOverloadReturnsThenable checks if a call expression's callee has any
// overload signature that returns a thenable type. This suppresses false
// positives when the type checker resolves to the wrong overload (common
// with intersection-typed functions like tar's extract).
func anyOverloadReturnsThenable(ctx rule.RuleContext, node *ast.Node) bool {
	expr := ast.SkipParentheses(node)
	if !ast.IsCallExpression(expr) {
		return false
	}
	callee := expr.AsCallExpression().Expression
	calleeType := ctx.TypeChecker.GetTypeAtLocation(callee)
	if calleeType == nil {
		return false
	}
	for _, sig := range checker.Checker_getSignaturesOfType(ctx.TypeChecker, calleeType, checker.SignatureKindCall) {
		retType := checker.Checker_getReturnTypeOfSignature(ctx.TypeChecker, sig)
		if retType != nil && utils.IsThenableType(ctx.TypeChecker, node, retType) {
			return true
		}
	}
	return false
}

func buildAwaitMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "await",
		Description: "Unexpected `await` of a non-Promise (non-\"Thenable\") value.",
	}
}

func buildRemoveAwaitMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "removeAwait",
		Description: "Remove unnecessary `await`.",
	}
}

func buildForAwaitOfNonAsyncIterableMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "forAwaitOfNonAsyncIterable",
		Description: "Unexpected `for await...of` of a value that is not async iterable.",
	}
}

func buildConvertToOrdinaryForMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "convertToOrdinaryFor",
		Description: "Convert to an ordinary `for...of` loop.",
	}
}

func buildAwaitUsingOfNonAsyncDisposableMessage() rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "awaitUsingOfNonAsyncDisposable",
		Description: "Unexpected `await using` of a value that is not async disposable.",
	}
}

var AwaitThenableRule = rule.Rule{
	Name: "await-thenable",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindAwaitExpression: func(node *ast.Node) {
				if node == nil {
					return
				}
				awaitExpr := node.AsAwaitExpression()
				if awaitExpr == nil || awaitExpr.Expression == nil {
					return
				}
				awaitArgument := awaitExpr.Expression
				awaitArgumentType := ctx.TypeChecker.GetTypeAtLocation(awaitArgument)
				if awaitArgumentType == nil {
					return
				}
				certainty := utils.NeedsToBeAwaited(ctx.TypeChecker, awaitArgument, awaitArgumentType)

			if certainty == utils.TypeAwaitableNever {
				// Skip if any overload of the called function returns a thenable.
				// The type checker may have resolved to the wrong overload.
				if anyOverloadReturnsThenable(ctx, awaitArgument) {
					return
				}
				awaitTokenRange := scanner.GetRangeOfTokenAtPosition(ctx.SourceFile, node.Pos())
					ctx.ReportRangeWithSuggestions(awaitTokenRange, buildAwaitMessage(), func() []rule.RuleSuggestion {
						return []rule.RuleSuggestion{{
							Message: buildRemoveAwaitMessage(),
							FixesArr: []rule.RuleFix{
								rule.RuleFixRemoveRange(awaitTokenRange),
							},
						}}
					})
				}
			},
			ast.KindForOfStatement: func(node *ast.Node) {
				if node == nil {
					return
				}
				stmt := node.AsForInOrOfStatement()
				if stmt == nil || stmt.AwaitModifier == nil {
					return
				}
				if stmt.Expression == nil {
					return
				}

				exprType := ctx.TypeChecker.GetTypeAtLocation(stmt.Expression)
				if exprType == nil || utils.IsTypeAnyType(exprType) {
					return
				}

				for _, typePart := range utils.UnionTypeParts(exprType) {
					if typePart == nil {
						continue
					}
					if utils.GetWellKnownSymbolPropertyOfType(typePart, "asyncIterator", ctx.TypeChecker) != nil {
						return
					}
				}

				ctx.ReportRangeWithSuggestions(
					utils.GetForStatementHeadLoc(ctx.SourceFile, node),
					buildForAwaitOfNonAsyncIterableMessage(),
					func() []rule.RuleSuggestion {
						// Note that this suggestion causes broken code for sync iterables
						// of promises, since the loop variable is not awaited.
						return []rule.RuleSuggestion{{
							Message: buildConvertToOrdinaryForMessage(),
							FixesArr: []rule.RuleFix{
								rule.RuleFixRemove(ctx.SourceFile, stmt.AwaitModifier),
							},
						}}
					},
				)
			},
			ast.KindVariableDeclarationList: func(node *ast.Node) {
				if node == nil || !ast.IsVarAwaitUsing(node) {
					return
				}

				declaration := node.AsVariableDeclarationList()
				if declaration == nil {
					return
				}
			DeclaratorLoop:
				for _, declarator := range declaration.Declarations.Nodes {
					init := declarator.Initializer()
					if init == nil {
						continue
					}
					initType := ctx.TypeChecker.GetTypeAtLocation(init)
					if initType == nil || utils.IsTypeAnyType(initType) {
						continue
					}

					for _, typePart := range utils.UnionTypeParts(initType) {
						if typePart == nil {
							continue
						}
						if utils.GetWellKnownSymbolPropertyOfType(typePart, "asyncDispose", ctx.TypeChecker) != nil {
							continue DeclaratorLoop
						}
					}

					var suggestions []rule.RuleSuggestion
					// let the user figure out what to do if there's
					// await using a = b, c = d, e = f;
					// it's rare and not worth the complexity to handle.
					if len(declaration.Declarations.Nodes) == 1 {
						suggestions = append(suggestions, rule.RuleSuggestion{
							Message: buildRemoveAwaitMessage(),
							FixesArr: []rule.RuleFix{
								rule.RuleFixRemoveRange(scanner.GetRangeOfTokenAtPosition(ctx.SourceFile, node.Pos())),
							},
						})
					}

					ctx.ReportNodeWithSuggestions(init, buildAwaitUsingOfNonAsyncDisposableMessage(), func() []rule.RuleSuggestion { return suggestions })
				}
			},
		}
	},
}
