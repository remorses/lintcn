// Resolve the .lintcn/ directory by walking up from cwd.
// This lets users run `lintcn lint` from any subdirectory of their project.

import { findUpSync } from 'find-up'
import path from 'node:path'

/** Find the nearest .lintcn/ directory by walking up from cwd.
 *  Returns the absolute path to the directory, or null if not found. */
export function findLintcnDir(): string | null {
  const found = findUpSync('.lintcn', { type: 'directory' })
  return found ?? null
}

/** Find .lintcn/ or throw with a helpful error. */
export function getLintcnDir(): string {
  const dir = findLintcnDir()
  if (dir) {
    return dir
  }
  // fall back to cwd/.lintcn for `lintcn add` (creates the directory)
  return path.resolve(process.cwd(), '.lintcn')
}

/** Find .lintcn/ or throw — for commands that require it to exist. */
export function requireLintcnDir(): string {
  const dir = findLintcnDir()
  if (!dir) {
    throw new Error('No .lintcn/ directory found in current or parent directories. Run `lintcn add <url>` first.')
  }
  return dir
}
