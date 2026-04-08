// Content hash for binary caching (local + remote).
// Combines cache schema version, tsgolint version, platform triplet,
// and non-test rule file contents into a SHA-256 hash.
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

const CACHE_SCHEMA_VERSION = '4'

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

  // walk rule subfolders for non-test .go files in sorted order
  const entries = fs.readdirSync(lintcnDir, { withFileTypes: true })
    .filter((e) => { return e.isDirectory() && !e.name.startsWith('.') })
    .sort((a, b) => { return a.name.localeCompare(b.name) })

  for (const entry of entries) {
    const subDir = path.join(lintcnDir, entry.name)
    const goFiles = fs.readdirSync(subDir)
      .filter((f) => { return f.endsWith('.go') && !f.endsWith('_test.go') })
      .sort()

    for (const file of goFiles) {
      const content = fs.readFileSync(path.join(subDir, file), 'utf-8')
      hash.update(`file:${entry.name}/${file}\n`)
      hash.update(content)
    }
  }

  const full = hash.digest('hex')
  return { short: full.slice(0, 16), full }
}
