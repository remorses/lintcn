// lintcn lint — build a custom tsgolint binary and run it against the project.
// Handles Go workspace generation, compilation with caching, and execution.

import fs from 'node:fs'
import path from 'node:path'
import { spawn } from 'node:child_process'
import { requireLintcnDir } from '../paths.ts'
import { discoverRules, type RuleMetadata } from '../discover.ts'
import { generateBuildWorkspace, generateEditorGoFiles } from '../codegen.ts'
import { ensureTsgolintSource, validateVersion, cachedBinaryExists, getBinaryPath, getBuildDir, getBinDir } from '../cache.ts'
import { computeContentHash } from '../hash.ts'
import { execAsync } from '../exec.ts'

async function checkGoInstalled(): Promise<void> {
  try {
    await execAsync('go', ['version'])
  } catch {
    throw new Error(
      'Go is required to build rules.\n' +
      'Install from https://go.dev/dl/',
    )
  }
}

export async function buildBinary({
  rebuild,
  tsgolintVersion,
}: {
  rebuild: boolean
  tsgolintVersion: string
}): Promise<string> {
  validateVersion(tsgolintVersion)
  await checkGoInstalled()

  const lintcnDir = requireLintcnDir()

  const rules = discoverRules(lintcnDir)
  if (rules.length === 0) {
    throw new Error('No rules found in .lintcn/. Run `lintcn add <url>` to add rules.')
  }

  console.log(`Found ${rules.length} custom rule${rules.length === 1 ? '' : 's'} (tsgolint ${tsgolintVersion.slice(0, 8)})`)

  // ensure tsgolint source
  const tsgolintDir = await ensureTsgolintSource(tsgolintVersion)

  // compute content hash
  const { short: contentHash } = await computeContentHash({
    lintcnDir,
    tsgolintVersion,
  })

  // check cache
  if (!rebuild && cachedBinaryExists(contentHash)) {
    console.log('Using cached binary')
    return getBinaryPath(contentHash)
  }

  // ensure .lintcn/go.mod exists (gitignored, needed by the build workspace symlink)
  generateEditorGoFiles(lintcnDir)

  // generate build workspace (per-hash dir to avoid races between concurrent processes)
  const buildDir = getBuildDir(contentHash)
  console.log('Generating build workspace...')
  generateBuildWorkspace({
    buildDir,
    tsgolintDir,
    lintcnDir,
    rules,
  })

  // compile
  const binDir = getBinDir()
  fs.mkdirSync(binDir, { recursive: true })
  const binaryPath = getBinaryPath(contentHash)

  // Check if any lintcn binary has been built before — if not, this is a cold
  // build that compiles the full tsgolint + typescript-go dependency tree.
  const existingBins = fs.existsSync(binDir) ? fs.readdirSync(binDir) : []
  if (existingBins.length === 0) {
    console.log('Compiling custom tsgolint binary (first build — may take 30s+ to compile dependencies)...')
    console.log('Subsequent builds will be fast (~1s). In CI, cache ~/.cache/lintcn/ and GOCACHE (run `go env GOCACHE`).')
  } else {
    console.log('Compiling custom tsgolint binary...')
  }

  const { exitCode: buildExitCode } = await execAsync('go', ['build', '-trimpath', '-o', binaryPath, './wrapper'], {
    cwd: buildDir,
    stdio: 'inherit',
  })
  if (buildExitCode !== 0) {
    throw new Error(`Go compilation failed (exit code ${buildExitCode})`)
  }

  console.log('Build complete')
  return binaryPath
}

export async function lint({
  rebuild,
  tsgolintVersion,
  passthroughArgs,
  allWarnings,
}: {
  rebuild: boolean
  tsgolintVersion: string
  passthroughArgs: string[]
  allWarnings: boolean
}): Promise<number> {
  const binaryPath = await buildBinary({ rebuild, tsgolintVersion })

  // Discover rules to inject --warn flags for warning-severity rules.
  // buildBinary already discovered rules for compilation, but we need the
  // metadata here to know which rules are warnings at runtime.
  const lintcnDir = requireLintcnDir()
  const rules = discoverRules(lintcnDir)
  const warnArgs = buildWarnArgs(rules)

  // By default, limit warnings to git-changed files so they don't flood
  // the output in large codebases. --all-warnings bypasses this filter.
  const hasWarnRules = rules.some((r) => r.severity === 'warn')
  let warnFileArgs: string[] = []
  if (hasWarnRules && allWarnings) {
    warnFileArgs = ['--all-warnings']
  } else if (hasWarnRules) {
    warnFileArgs = await buildWarnFileArgs()
  }

  // run the binary with --warn + --warn-file/--all-warnings flags + passthrough args
  const allArgs = [...warnArgs, ...warnFileArgs, ...passthroughArgs]
  return new Promise((resolve) => {
    const proc = spawn(binaryPath, allArgs, {
      stdio: 'inherit',
    })

    proc.on('error', (err) => {
      console.error(`Failed to run binary: ${err.message}`)
      resolve(1)
    })

    proc.on('close', (code) => {
      resolve(code ?? 1)
    })
  })
}

/** Build --warn flags for rules with severity 'warn'.
 *  Uses goRuleName (parsed from Go source) to match the runtime name
 *  that tsgolint uses in diagnostics, avoiding silent mismatches. */
function buildWarnArgs(rules: RuleMetadata[]): string[] {
  const args: string[] = []
  for (const rule of rules) {
    if (rule.severity === 'warn') {
      args.push('--warn', rule.goRuleName)
    }
  }
  return args
}

/** Get git-changed files and build --warn-file flags so warnings only
 *  appear for new/modified code. Returns [] if git is unavailable or not
 *  a git repo — the runner will then show no warnings (safe default).
 *  Linting must never crash from this. */
async function buildWarnFileArgs(): Promise<string[]> {
  try {
    // Get git repo root to resolve relative paths to absolute.
    const topLevelResult = await execAsync('git', ['rev-parse', '--show-toplevel'], { stdio: 'pipe' }).catch(() => null)
    if (!topLevelResult) return []
    const repoRoot = topLevelResult.stdout.trim()

    // Changed files (staged + unstaged vs HEAD)
    const diffResult = await execAsync('git', ['diff', '--name-only', 'HEAD'], { stdio: 'pipe' }).catch(() => null)
    // Untracked files (new files not yet committed)
    const untrackedResult = await execAsync('git', ['ls-files', '--others', '--exclude-standard'], { stdio: 'pipe' }).catch(() => null)

    const files = new Set<string>()

    for (const result of [diffResult, untrackedResult]) {
      if (!result) continue
      for (const line of result.stdout.split('\n')) {
        const trimmed = line.trim()
        if (trimmed) {
          // Resolve to absolute path so it matches SourceFile.FileName() in the runner.
          files.add(path.resolve(repoRoot, trimmed))
        }
      }
    }

    // No changed files → no --warn-file flags → runner shows no warnings (clean tree)
    if (files.size === 0) return []

    const args: string[] = []
    for (const file of files) {
      args.push('--warn-file', file)
    }
    return args
  } catch {
    // git not installed, not a repo, or any other failure — no warnings shown
    return []
  }
}
