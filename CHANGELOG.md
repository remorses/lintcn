## 0.4.0

1. **Simpler rule imports** — rules now import from `pkg/rule`, `pkg/utils`, etc. instead of `internal/rule`. The `internal/` child-module-path hack is gone. Your `.go` files use clean import paths:
   ```go
   import (
       "github.com/typescript-eslint/tsgolint/pkg/rule"
       "github.com/typescript-eslint/tsgolint/pkg/utils"
   )
   ```

2. **Simpler codegen** — uses a tsgolint fork with `pkg/runner.Run()`, eliminating all regex surgery on main.go. The generated binary entry point is a 15-line template instead of a patched copy of tsgolint's main.go.

## 0.3.0

1. **Only custom rules run by default** — previously the binary included all 44 built-in tsgolint rules, producing thousands of noisy errors. Now only your `.lintcn/` rules run. True shadcn model: explicitly add each rule you want.

   Before (0.2.0): `Found 2315 errors (linted 193 files with 45 rules)`
   After (0.3.0): `Found 8 errors (linted 193 files with 1 rule)`

2. **Run `lintcn lint` from any subdirectory** — uses `find-up` to walk parent directories for `.lintcn/`. You no longer need to be at the project root:
   ```bash
   cd packages/my-app
   lintcn lint   # finds .lintcn/ in parent
   ```

3. **No git required** — tsgolint source is now downloaded as a tarball from GitHub instead of cloned. Patches applied with `patch -p1`. Faster first setup, works without git installed.

4. **Fixed stale binary cache** — added `CACHE_SCHEMA_VERSION` to the content hash. Upgrading lintcn now correctly invalidates cached binaries built by older versions.

5. **Fixed partial download corruption** — if the tsgolint download fails midway, the partial directory is cleaned up so the next run starts fresh instead of failing repeatedly.

6. **Fixed GitHub URLs with `/` in branch names** — `lintcn add` now correctly handles branch names like `feature/my-branch` in GitHub blob URLs.

## 0.2.0

1. **Pinned tsgolint version** — each lintcn release bundles a specific tsgolint version (`v0.9.2`). Builds are now reproducible: everyone on the same lintcn version compiles against the same tsgolint API. Previously used `main` branch which was non-deterministic.

2. **`--tsgolint-version` flag** — override the pinned version for testing unreleased tsgolint:
   ```bash
   npx lintcn lint --tsgolint-version v0.10.0
   ```

3. **Version pinning docs** — README now explains why you should pin lintcn in `package.json` (no `^` or `~`) and how to update safely.

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
