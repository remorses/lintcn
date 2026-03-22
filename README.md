# lintcn

The [shadcn](https://ui.shadcn.com) for type-aware TypeScript lint rules. Powered by [tsgolint](https://github.com/oxc-project/tsgolint).

Add rules by URL, own the source, customize freely. Rules are Go files that use the TypeScript type checker for deep analysis — things ESLint can't do.

## Install

```bash
npm install -D lintcn
```

## Usage

```bash
# Add a rule folder from tsgolint
npx lintcn add https://github.com/oxc-project/tsgolint/tree/main/internal/rules/no_floating_promises

# Add by file URL (auto-fetches the whole folder)
npx lintcn add https://github.com/oxc-project/tsgolint/blob/main/internal/rules/await_thenable/await_thenable.go

# Lint your project
npx lintcn lint

# Lint with a specific tsconfig
npx lintcn lint --tsconfig tsconfig.build.json

# List installed rules
npx lintcn list

# Remove a rule
npx lintcn remove no-floating-promises

# Clean cached tsgolint source + binaries
npx lintcn clean
```

Browse all 50+ available built-in rules in the [tsgolint rules directory](https://github.com/oxc-project/tsgolint/tree/main/internal/rules).

## How it works

Each rule lives in its own subfolder under `.lintcn/`. You own the source — edit, customize, delete.

```
my-project/
├── .lintcn/
│   ├── .gitignore                          ← ignores generated Go files
│   ├── no_floating_promises/
│   │   ├── no_floating_promises.go         ← rule source (committed)
│   │   ├── no_floating_promises_test.go    ← tests (committed)
│   │   ├── options.go                      ← rule options struct
│   │   └── schema.json                     ← options schema
│   ├── await_thenable/
│   │   ├── await_thenable.go
│   │   └── await_thenable_test.go
│   └── my_custom_rule/
│       └── my_custom_rule.go
├── src/
│   └── ...
├── tsconfig.json
└── package.json
```

When you run `npx lintcn lint`, the CLI:

1. Scans `.lintcn/*/` subfolders for rule definitions
2. Generates a Go workspace with your custom rules
3. Compiles a custom binary (cached — rebuilds only when rules change)
4. Runs the binary against your project

You can run `lintcn lint` from any subdirectory — it walks up to find `.lintcn/` and lints the cwd project.

## Writing custom rules

To help AI agents write and modify rules, install the lintcn skill:

```bash
npx skills add remorses/lintcn
```

This gives your AI agent the full tsgolint rule API reference — AST visitors, type checker, reporting, fixes, and testing patterns.

Every rule lives in a subfolder under `.lintcn/` with the package name matching the folder:

```go
// .lintcn/no_unhandled_error/no_unhandled_error.go

// lintcn:name no-unhandled-error
// lintcn:description Disallow discarding Error-typed return values

package no_unhandled_error

import (
    "github.com/microsoft/typescript-go/shim/ast"
    "github.com/microsoft/typescript-go/shim/checker"
    "github.com/typescript-eslint/tsgolint/internal/rule"
    "github.com/typescript-eslint/tsgolint/internal/utils"
)

var NoUnhandledErrorRule = rule.Rule{
    Name: "no-unhandled-error",
    Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
        return rule.RuleListeners{
            ast.KindExpressionStatement: func(node *ast.Node) {
                expression := ast.SkipParentheses(node.AsExpressionStatement().Expression)

                if ast.IsVoidExpression(expression) {
                    return // void = intentional discard
                }

                innerExpr := expression
                if ast.IsAwaitExpression(innerExpr) {
                    innerExpr = ast.SkipParentheses(innerExpr.Expression())
                }
                if !ast.IsCallExpression(innerExpr) {
                    return
                }

                t := ctx.TypeChecker.GetTypeAtLocation(expression)

                if utils.IsTypeFlagSet(t, checker.TypeFlagsVoid|checker.TypeFlagsUndefined|checker.TypeFlagsNever) {
                    return
                }

                for _, part := range utils.UnionTypeParts(t) {
                    if utils.IsErrorLike(ctx.Program, ctx.TypeChecker, part) {
                        ctx.ReportNode(node, rule.RuleMessage{
                            Id:          "noUnhandledError",
                            Description: "Error-typed return value is not handled.",
                        })
                        return
                    }
                }
            },
        }
    },
}
```

This catches code like:

```typescript
// error — result discarded, Error not handled
getUser("id")           // returns Error | User
await fetchData("/api") // returns Promise<Error | Data>

// ok — result is checked
const user = getUser("id")
if (user instanceof Error) return user

// ok — explicitly discarded
void getUser("id")
```

## Version pinning

**Pin lintcn in your `package.json`** — do not use `^` or `~`:

```json
{
  "devDependencies": {
    "lintcn": "0.5.0"
  }
}
```

Each lintcn release bundles a specific tsgolint version. Updating lintcn can change the underlying tsgolint API, which may cause your rules to no longer compile. Always update consciously:

1. Check the [changelog](./CHANGELOG.md) for tsgolint version changes
2. Run `npx lintcn build` after updating to verify your rules still compile
3. Fix any compilation errors before committing

## Prerequisites

- **Node.js** — for the CLI
- **Go** — for compiling rules (`go.dev/dl`)

Go is only needed for `lintcn lint` / `lintcn build`. Adding and listing rules works without Go.

## License

MIT
