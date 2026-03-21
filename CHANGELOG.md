## 0.1.0

1. **Initial release** — CLI for adding type-aware TypeScript lint rules as Go files to your project:

   ```bash
   npx lintcn add https://github.com/user/repo/blob/main/rules/no_unhandled_error.go
   npx lintcn lint
   ```

2. **`lintcn add <url>`** — fetch a `.go` rule file by URL into `.lintcn/`. Normalizes GitHub blob URLs to raw URLs automatically. Also fetches the matching `_test.go` if present. Rewrites the package declaration to `package lintcn` and injects a `// lintcn:source` comment.

3. **`lintcn lint`** — builds a custom tsgolint binary (all 50+ built-in rules + your custom rules) and runs it against the project. Binary is cached by SHA-256 content hash — rebuilds only when rules change.

4. **`lintcn build`** — build the custom binary without running it. Prints the binary path.

5. **`lintcn list`** — list installed rules with descriptions parsed from `// lintcn:` metadata comments.

6. **`lintcn remove <name>`** — delete a rule and its test file from `.lintcn/`.

7. **Editor/LSP support** — generates `go.work` and `go.mod` inside `.lintcn/` so gopls provides full autocomplete, go-to-definition, and type checking on tsgolint APIs while writing rules.
