// Content hash for binary caching (local + remote).
// Combines cache schema version, tsgolint version, platform triplet,
// and rule source/asset contents into a SHA-256 hash.
//
// The hash is deterministic across machines — same rules + same tsgolint
// version + same platform = same hash. Go version is NOT included because
// the compiled binary is standalone (no Go runtime dependency).
//
// Bump CACHE_SCHEMA_VERSION when codegen logic changes to invalidate
// stale binaries built by older lintcn versions.

import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'

const CACHE_SCHEMA_VERSION = '5'

/** Compute a deterministic content hash for binary caching.
 *  Returns: { short: "a1b2c3d4..." (16 hex), full: "a1b2c3d4..." (64 hex) }
 *  The short hash is used for local cache paths, the full hash for remote cache keys. */
export async function computeContentHash({
  lintcnDir,
  tsgolintVersion,
}: {
  lintcnDir: string
  tsgolintVersion: string
}): Promise<{ short: string; full: string }> {
  const hash = crypto.createHash('sha256')

  hash.update(`cache-schema:${CACHE_SCHEMA_VERSION}\n`)
  hash.update(`tsgolint:${tsgolintVersion}\n`)
  hash.update(`platform:${process.platform}-${process.arch}\n`)

  for (const file of collectRuleFiles(lintcnDir).sort()) {
    const content = fs.readFileSync(path.join(lintcnDir, file))
    // Frame both the path and raw bytes so file boundaries are unambiguous.
    hash.update(`file:${JSON.stringify(file)}\nsize:${content.length}\n`)
    hash.update(content)
  }

  const full = hash.digest('hex')
  return { short: full.slice(0, 16), full }
}

// These are lintcn's generated files, not inputs to the custom binary.
const GENERATED_ROOT_ENTRIES = new Set(['.tsgolint', '.gitignore', 'go.mod', 'go.sum', 'go.work', 'go.work.sum'])
const VCS_ENTRIES = new Set(['.git', '.hg', '.svn'])

/** Include assets conservatively instead of trying to parse go:embed patterns.
 *  A cache lookup must work without invoking Go, including for binary assets. */
function collectRuleFiles(lintcnDir: string): string[] {
  const files: string[] = []
  const ancestors = new Set<string>()
  function visit(directory: string, relative: string): void {
    const realDirectory = fs.realpathSync(directory)
    if (ancestors.has(realDirectory)) {
      throw new Error(`Circular symbolic link in lintcn sources: ${relative}`)
    }
    ancestors.add(realDirectory)
    try {
      for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
        if (!relative && GENERATED_ROOT_ENTRIES.has(entry.name)) continue
        if (VCS_ENTRIES.has(entry.name)) continue
        const file = relative ? `${relative}/${entry.name}` : entry.name
        const absolute = path.join(directory, entry.name)
        const stat = entry.isSymbolicLink() ? fs.statSync(absolute) : entry
        if (stat.isDirectory()) {
          if (entry.name !== '__snapshots__') visit(absolute, file)
        } else if (stat.isFile() && !entry.name.endsWith('_test.go')) {
          files.push(file)
        }
      }
    } finally {
      ancestors.delete(realDirectory)
    }
  }
  visit(lintcnDir, '')
  return files
}
