# lintcn

## tsgolint fork dependency

lintcn depends on a fork of tsgolint at `remorses/tsgolint`. The fork has
2 commits on top of upstream (`oxc-project/tsgolint`):

1. `910200c` — moves `internal/{rule,utils,linter,diagnostic,rule_tester,collections}`
   to `pkg/` and updates all imports. Adds `pkg/runner` with `Run(rules, args)`.
2. `a936043` — copies fixtures to `pkg/rule_tester/fixtures/` for external test access.

These 2 commits are the only difference from upstream. Everything else is
identical.

### version pinning

Two constants in `src/cache.ts` control which versions are downloaded:

- `DEFAULT_TSGOLINT_VERSION` — commit hash from `remorses/tsgolint`
- `TYPESCRIPT_GO_COMMIT` — commit hash from `microsoft/typescript-go`
  (the base commit before tsgolint's patches are applied)

Both must be updated together when syncing with upstream.

### updating the fork with upstream

The fork lives at `/Users/morse/Documents/GitHub/tsgolint`. To sync:

```bash
cd /Users/morse/Documents/GitHub/tsgolint

# add upstream if not already added
git remote add upstream https://github.com/oxc-project/tsgolint.git

# fetch upstream changes
git fetch upstream

# merge upstream into our fork (prefer merge over rebase to keep history clean)
git merge upstream/main

# if there are conflicts in internal/rules/ imports (upstream added new rules
# that still import from internal/), fix them: the rule files should import
# from pkg/ not internal/. run this to fix:
find internal/rules -name '*.go' -exec sed -i '' 's|tsgolint/internal/rule"|tsgolint/pkg/rule"|g' {} +
find internal/rules -name '*.go' -exec sed -i '' 's|tsgolint/internal/utils"|tsgolint/pkg/utils"|g' {} +
find internal/rules -name '*.go' -exec sed -i '' 's|tsgolint/internal/rule_tester"|tsgolint/pkg/rule_tester"|g' {} +
find internal/rules -name '*.go' -exec sed -i '' 's|tsgolint/internal/diagnostic"|tsgolint/pkg/diagnostic"|g' {} +

# if upstream updated any pkg/ that we also have in pkg/, merge those changes too.
# the shared packages (rule, utils, linter, etc.) might have new functions or
# changed signatures. copy the updated files:
cp -r internal/rule/* pkg/rule/ 2>/dev/null   # if internal/rule still exists
cp -r internal/utils/* pkg/utils/ 2>/dev/null
cp -r internal/linter/* pkg/linter/ 2>/dev/null
cp -r internal/diagnostic/* pkg/diagnostic/ 2>/dev/null
cp -r internal/collections/* pkg/collections/ 2>/dev/null

# wait — upstream doesn't have internal/rule anymore (we deleted it).
# so if upstream added new files to internal/rule, they'll appear as
# new files in the merge. the correct approach:
#
# 1. after merge, check if internal/{rule,utils,linter,...} reappeared
# 2. if yes, upstream added new files. move them to pkg/ instead
# 3. delete the internal/ copies (we only keep internal/rules/)

# verify build
go build ./...

# update typescript-go submodule if upstream changed it
git submodule update --init

# apply patches (upstream may have added/changed patches)
cd typescript-go
git am --3way --no-gpg-sign ../patches/*.patch
cd ..

# copy collections
mkdir -p internal/collections
find typescript-go/internal/collections -name '*.go' ! -name '*_test.go' -exec cp {} internal/collections/ \;

# copy fixtures if upstream added new ones
cp -r internal/rules/fixtures/* pkg/rule_tester/fixtures/ 2>/dev/null

# verify build again
go build ./...

# push
git push origin main --no-recurse-submodules
```

### updating lintcn after fork sync

After pushing the updated fork:

```bash
cd /Users/morse/Documents/GitHub/tsgolint

# get the new commit hash
git rev-parse HEAD
# → abc123...

# get the new typescript-go base commit (first commit before patches)
cd typescript-go
git log --oneline --reverse | head -1
# → 1b7eabe1 fix: nil pointer deref...
# use this hash (the one BEFORE any patch commits)
cd ..
```

Then update `lintcn/src/cache.ts`:

```ts
export const DEFAULT_TSGOLINT_VERSION = '<new fork commit hash>'
const TYPESCRIPT_GO_COMMIT = '<new typescript-go base commit>'
```

The typescript-go commit changes only when upstream tsgolint updates its
submodule. Check `git diff upstream/main -- .gitmodules` or
`git ls-tree HEAD typescript-go` to see if it changed.

After updating:

```bash
cd lintcn

# clear cache to test fresh download
rm -rf ~/.cache/lintcn

# build and test
pnpm build
cd ../discord && node ../lintcn/dist/cli.js lint

# bump CACHE_SCHEMA_VERSION in src/hash.ts if the fork changed
# pkg/runner or codegen-affecting code
```

### what the typescript-go dependency is

tsgolint depends on `microsoft/typescript-go` — the Go port of the TypeScript
compiler. tsgolint uses it for AST parsing, type checking, and program creation.

The typescript-go version is managed entirely by the tsgolint repo:
- tsgolint's `.gitmodules` points to `microsoft/typescript-go`
- tsgolint pins a specific commit of typescript-go as a git submodule
- tsgolint applies patches on top (in `patches/`) for features it needs
- when tsgolint updates typescript-go, they update the submodule + patches

lintcn does NOT independently track typescript-go. It follows whatever
version tsgolint uses. The `TYPESCRIPT_GO_COMMIT` constant in cache.ts
is just the base commit from tsgolint's submodule — it changes only when
tsgolint updates its submodule.

### when to update

- **tsgolint adds new rules** — sync fork, update hashes, publish new lintcn
  (users can `lintcn add` the new rules)
- **tsgolint changes pkg/rule API** — sync fork, update hashes, bump
  CACHE_SCHEMA_VERSION, update SKILL.md, publish new lintcn (may break
  user rules — changelog must warn)
- **tsgolint updates typescript-go** — sync fork, update both hashes,
  publish new lintcn (usually transparent to users)
- **security fix in tsgolint** — sync fork urgently

### what NOT to do

- never fork microsoft/typescript-go — it updates too frequently
- never modify typescript-go source directly — only use tsgolint's patches
- never rebase the fork — merge upstream to keep history clean
- never change the fork's go.mod module path — it must stay
  `github.com/typescript-eslint/tsgolint` for import compatibility
