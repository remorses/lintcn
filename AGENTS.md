# lintcn

## tsgolint fork

lintcn uses `remorses/tsgolint` (forked from `oxc-project/tsgolint`).

The fork adds on top of upstream:
1. `internal/runner/runner.go` — new file with `Run(rules, args)` entry point
2. `internal/rule_tester/snapshot.go` — added `TSGOLINT_SNAPSHOT_CWD` env var.
   When `TSGOLINT_SNAPSHOT_CWD=true`, snapshots are stored relative to
   `os.Getwd()` (the test package directory) instead of relative to snapshot.go.
   Default behavior is unchanged — tsgolint's own snapshots stay in
   `internal/rule_tester/__snapshots__/`.

User rules must set `TSGOLINT_SNAPSHOT_CWD=true` when running `go test` so
snapshots land in `.lintcn/<rule>/__snapshots__/` instead of the cached
tsgolint source directory.

User rules import from `internal/rule`, `internal/utils` etc. — same paths
as tsgolint's own code. The Go workspace allows this because the user module
name is a child path: `github.com/typescript-eslint/tsgolint/lintcn-rules`.

## updating tsgolint

Two constants in `src/cache.ts`:

- `DEFAULT_TSGOLINT_VERSION` — commit hash from `remorses/tsgolint`
- `TYPESCRIPT_GO_COMMIT` — base commit from `microsoft/typescript-go`
  (before patches). Changes only when upstream updates its submodule.

To sync: merge upstream into fork, push, update both constants, clear
`~/.cache/lintcn`, rebuild, test.

typescript-go is managed by tsgolint — never fork it independently.
