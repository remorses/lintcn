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

# Show warning rules for all files, not just updated/added files
npx lintcn lint --all-warnings

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
│   │   └── options.go                      ← rule options struct
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
getUser("id"); // returns Error | User
await fetchData("/api"); // returns Promise<Error | Data>

// ok — result is checked
const user = getUser("id");
if (user instanceof Error) return user;

// ok — explicitly discarded
void getUser("id");
```

## Warning severity

Rules can be configured as **warnings** instead of errors:

- **Don't fail CI** — warnings produce exit code 0
- **Only shown for updated/added files by default** — warning rules are limited to files in `git diff` plus untracked files, so unchanged files are silently skipped

This lets you adopt new rules gradually. In a large codebase, enabling a rule as an error means hundreds of violations at once. As a warning, you only see violations in files you're actively changing or adding — fixing issues in new code without blocking the build.

### Configuring a rule as a warning

Add `// lintcn:severity warn` to the rule's Go file:

```go
// lintcn:name no-unhandled-error
// lintcn:severity warn
// lintcn:description Disallow discarding Error-typed return values
```

Rules without `// lintcn:severity` default to `error`.

### When warnings are shown

By default, `lintcn lint` runs `git diff` to find updated files and also includes untracked files you just added. Warning rules are only printed for files in that set:

```bash
# Warnings only for updated files plus newly added untracked files (default)
npx lintcn lint

# Warnings for ALL files, ignoring git diff
npx lintcn lint --all-warnings
```

| Scenario                                  | Warnings shown?   |
| ----------------------------------------- | ----------------- |
| File is updated in `git diff`             | Yes               |
| File is newly added and untracked         | Yes               |
| File is committed and unchanged           | No                |
| `--all-warnings` flag is passed           | Yes, all files    |
| Git is not installed or not a repo        | No warnings shown |
| Clean git tree (no changes, no new files) | No warnings shown |

### Workflow

1. Add a new rule with `lintcn add`
2. Set it to `// lintcn:severity warn` in the Go source
3. Run `lintcn lint` — only see warnings in files you're currently editing or adding
4. Fix warnings as you touch files naturally
5. Once the codebase is clean, change to `// lintcn:severity error` (or remove the directive) to enforce it

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

## CI Setup

The first `lintcn lint` compiles a custom Go binary (~30s). Subsequent runs use the cached binary (<1s). A binary cache hit needs only Node.js: it does not check for Go or download compiler sources. Cache `~/.cache/lintcn/` and Go's build cache to keep CI fast.

```yaml
# .github/workflows/lint.yml
name: Lint
on: [push, pull_request]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: 22

      - name: Cache lintcn binary + Go build cache
        uses: actions/cache@v4
        with:
          path: |
            ~/.cache/lintcn
            ~/go/pkg
          key: lintcn-${{ runner.os }}-${{ runner.arch }}-${{ hashFiles('.lintcn/**', 'package-lock.json') }}
          restore-keys: |
            lintcn-${{ runner.os }}-${{ runner.arch }}-

      - run: npm ci
      - run: npx lintcn lint
```

The binary cache key includes nested Go helpers and assets of any file type, including files used by `//go:embed`. File paths and raw bytes are hashed in a stable order. Changing, adding, deleting or renaming an asset invalidates the binary cache.

The hash excludes `*_test.go`, `__snapshots__`, version-control directories, and lintcn's generated root entries: `.tsgolint`, `.gitignore`, `go.mod`, `go.sum`, `go.work`, and `go.work.sum`. Other files are included conservatively, so changing an unreferenced asset or a rule's README can also cause a rebuild. Source symlinks are followed; circular directory links are rejected.

The Actions cache key includes rule assets and the package lockfile so updated binaries can be saved after either changes. The `restore-keys` fallback preserves older build inputs for reuse.

## Prerequisites

- **Node.js** — for the CLI
- **Go** — for compiling rules (`go.dev/dl`)

Go is needed when `lintcn lint` / `lintcn build` must compile a binary, including explicit rebuilds. Cached binaries, adding rules, and listing rules work without Go.

## License

MIT
