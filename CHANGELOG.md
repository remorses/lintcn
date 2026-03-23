## 0.7.0

1. **`lintcn lint --fix`** — automatically apply fixes in-place. Collects diagnostics per file, applies fixes via the Go runner, and only reports what couldn't be auto-fixed:

   ```bash
   lintcn lint --fix
   ```

2. **Warning severity system** — rules can now declare `// lintcn:severity warn`. Warnings don't fail CI (exit 0) and are filtered to git-changed files by default so they don't flood large codebases:

   ```go
   // lintcn:severity warn
   ```

   Two new flags:
   - `--all-warnings` — show warnings for all files, not just changed ones
   - `lintcn list` now shows a `(warn)` suffix on warning-severity rules

3. **New rule: `no-type-assertion` (warn)** — flags every `as X` / `<X>expr` and includes the actual expression type so agents know what they're working with:

   ```
   warning: Type assertion to `User ({ name: string; age: number })` from `unknown`
   ```

   User-defined types show their structural form in parentheses. Standard library types (Array, Map, Promise, etc.) are not expanded. Assertion chains (`x as unknown as Foo`) walk back to the original source type. Casting from `any` is silently allowed (standard untyped-API pattern).

4. **New rule: `no-in-operator` (warn)** — warns on every `in` expression and shows the expanded type of the right-hand operand. When the right side is a union and the property exists in some members but not others, it names which members have it and suggests using a discriminant property instead:

   ```
   warning: Avoid the `in` operator on `Cat | Dog`. Property `meow` exists in Cat but not Dog.
   Consider using a discriminant property (e.g. `kind`) instead of `in`.
   ```

5. **New rule: `no-redundant-in-check` (error)** — flags `"y" in x` when the type already has `y` as a required non-optional property in all union members. The check can never be false — it's dead code:

   ```ts
   interface User { name: string; age: number }
   if ('name' in user) { ... }  // error: redundant — User always has 'name'
   ```

6. **New built-in rules**: `jsx-no-leaked-render`, `no-unhandled-error`, `no-useless-coalescing` — available via `lintcn add` or by pointing at the `.lintcn/` folder URL.

7. **`lintcn add` with whole repo URL** — download all rules from a repo's `.lintcn/` folder in one shot. Merge semantics: remote rule folders overwrite local ones; local-only rules are preserved:

   ```bash
   # bare repo URL — fetches all rules from .lintcn/ at repo root
   lintcn add https://github.com/remorses/lintcn

   # tree URL pointing at a .lintcn collection
   lintcn add https://github.com/remorses/lintcn/tree/main/.lintcn
   ```

8. **Fixed `await-thenable` false positive on overloaded functions** — when a function has multiple call signatures (overloads or intersection-of-callable-types), the rule now checks if any overload returns a thenable before reporting. Fixes false positives like `await extract({...})` from the `tar` package.

9. **Brighter error underline** — error highlights changed from ANSI 256-color 160 (muted red) to 196 (pure bright red). Run `lintcn clean` to clear the old cached binary.

## 0.6.0

1. **Rules now live in subfolders** — each rule is its own Go package under `.lintcn/{rule_name}/`, replacing the old flat `.lintcn/*.go` layout. This eliminates the need to rename `options.go` and `schema.json` companions — they stay in the subfolder with their original names, and the Go package name matches the folder. `lintcn add` now fetches the entire rule folder automatically.

   ```
   .lintcn/
       no_floating_promises/
           no_floating_promises.go
           no_floating_promises_test.go
           options.go      ← original name, no renaming
           schema.json
       my_custom_rule/
           my_custom_rule.go
   ```

2. **`lintcn add` fetches whole folders** — both folder URLs (`/tree/`) and file URLs (`/blob/`) now fetch every `.go` and `.json` file in the rule's directory. Passing a file URL auto-detects the parent folder:

   ```bash
   # folder URL
   lintcn add https://github.com/oxc-project/tsgolint/tree/main/internal/rules/no_floating_promises

   # file URL — auto-fetches the whole folder
   lintcn add https://github.com/oxc-project/tsgolint/blob/main/internal/rules/await_thenable/await_thenable.go
   ```

3. **Error for flat `.go` files in `.lintcn/`** — if leftover flat files from older versions are detected, lintcn now prints a clear migration error with instructions instead of silently ignoring them.

4. **Reproducible builds with `-trimpath`** — the Go binary is now built with `-trimpath`, stripping absolute paths from the output. Binaries are identical across machines for the same rule content + tsgolint version + platform.

5. **Faster cache hits** — Go version removed from the content hash. The compiled binary is a standalone executable with no Go runtime dependency, so the Go version used to build it doesn't affect correctness. Also excludes `_test.go` files from the hash since tests don't affect the binary.

6. **Go compilation output is live** — `go build` now inherits stdio, so compilation progress and errors stream directly to the terminal instead of being silently captured.

7. **First-build guidance** — on first compile (cold Go cache), lintcn explains the one-time 30s cost and shows which directories to cache in CI:
   ```
   Compiling custom tsgolint binary (first build — may take 30s+ to compile dependencies)...
   Subsequent builds will be fast (~1s). In CI, cache ~/.cache/lintcn/ and GOCACHE (run `go env GOCACHE`).
   ```

8. **GitHub Actions example** — README now includes a copy-paste workflow that caches the compiled binary. Subsequent CI runs take ~12s (vs ~4min cold):

   ```yaml
   - name: Cache lintcn binary + Go build cache
     uses: actions/cache@v4
     with:
       path: |
         ~/.cache/lintcn
         ~/go/pkg
       key: lintcn-${{ runner.os }}-${{ runner.arch }}-${{ hashFiles('.lintcn/**/*.go') }}
       restore-keys: lintcn-${{ runner.os }}-${{ runner.arch }}-
   ```

## 0.5.0

1. **Security fix — path traversal in `--tsgolint-version`** — the version flag is now validated against a strict pattern. Previously a value like `../../etc` could escape the cache directory.

2. **Fixed intermittent failures with concurrent `lintcn lint` runs** — build workspaces are now per-content-hash instead of shared. Two processes running simultaneously no longer corrupt each other's build.

3. **Cross-platform tar extraction** — replaced shell `tar` command with the npm `tar` package. Works on Windows without needing system tar.

4. **No more `patch` command required** — tsgolint downloads now use a fork with a clean `internal/runner.Run()` entry point. Zero modifications to existing tsgolint files; upstream syncs will never conflict.

5. **Downloads no longer hang** — 120s timeout on all GitHub tarball downloads.

6. **Fixed broken `.tsgolint` symlink** — `lintcn add` now correctly detects and recreates broken symlinks.

## 0.4.0

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
