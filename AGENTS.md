# lintcn

## tsgolint fork

lintcn uses `remorses/tsgolint` (forked from `oxc-project/tsgolint`).

The fork adds `pkg/runner.Run(rules, args)` — a public entry point that
accepts a rules slice and handles everything (tsconfig, TS program, linting,
output). This lets our generated main.go be a 15-line template instead of
a regex-patched copy of tsgolint's main.go.

The fork also moves `internal/{rule,utils,linter,diagnostic,rule_tester}`
to `pkg/` so user rules can import them without a Go module naming hack.

## updating tsgolint

Two constants in `src/cache.ts`:

- `DEFAULT_TSGOLINT_VERSION` — commit hash from `remorses/tsgolint`
- `TYPESCRIPT_GO_COMMIT` — base commit from `microsoft/typescript-go`
  (before tsgolint's patches). Changes only when upstream tsgolint
  updates its submodule.

To sync the fork with upstream:

1. `cd` to the tsgolint fork repo
2. `git fetch upstream && git merge upstream/main`
3. Fix any import conflicts: new upstream rules may import `internal/rule`
   instead of `pkg/rule` — sed replace them
4. If upstream added files to packages we moved to `pkg/`, move the new
   files too
5. `go build ./...` to verify
6. Push, get new commit hash, update both constants in `cache.ts`
7. Clear `~/.cache/lintcn`, rebuild, test

typescript-go is managed by tsgolint — we just download the base commit
tsgolint pins and apply tsgolint's patches. Never fork typescript-go.
