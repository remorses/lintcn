// Manage cached tsgolint source clone and compiled binaries.
// Cache lives in ~/.cache/lintcn/ with structure:
//   tsgolint/<version>/   — cloned tsgolint source (read-only)
//   bin/<content-hash>    — compiled binaries

import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { execAsync } from './exec.ts'

// Default tsgolint version — pinned to a known-good commit
export const DEFAULT_TSGOLINT_VERSION = 'main'

export function getCacheDir(): string {
  return path.join(os.homedir(), '.cache', 'lintcn')
}

export function getTsgolintSourceDir(version: string): string {
  return path.join(getCacheDir(), 'tsgolint', version)
}

export function getBinDir(): string {
  return path.join(getCacheDir(), 'bin')
}

export function getBinaryPath(contentHash: string): string {
  return path.join(getBinDir(), contentHash)
}

export function getBuildDir(): string {
  return path.join(getCacheDir(), 'build')
}

export async function ensureTsgolintSource(version: string): Promise<string> {
  const sourceDir = getTsgolintSourceDir(version)
  const readyMarker = path.join(sourceDir, '.lintcn-ready')

  if (fs.existsSync(readyMarker)) {
    return sourceDir
  }

  console.log(`Cloning tsgolint@${version}...`)

  fs.mkdirSync(sourceDir, { recursive: true })

  // clone with depth 1 for speed
  const cloneArgs = ['clone', '--depth', '1', '--recurse-submodules', '--shallow-submodules']
  if (version !== 'main') {
    cloneArgs.push('--branch', version)
  }
  cloneArgs.push('https://github.com/oxc-project/tsgolint.git', sourceDir)

  await execAsync('git', cloneArgs)

  // apply patches if they exist
  const patchesDir = path.join(sourceDir, 'patches')
  if (fs.existsSync(patchesDir)) {
    const patches = fs.readdirSync(patchesDir).filter((f) => {
      return f.endsWith('.patch')
    }).sort()

    if (patches.length > 0) {
      console.log(`Applying ${patches.length} patches...`)
      const patchPaths = patches.map((p) => {
        return path.join('..', 'patches', p)
      })
      await execAsync('git', ['am', '--3way', '--no-gpg-sign', ...patchPaths], {
        cwd: path.join(sourceDir, 'typescript-go'),
      })
    }
  }

  // copy internal/collections from typescript-go (required by tsgolint, done by `just init`)
  const collectionsDir = path.join(sourceDir, 'internal', 'collections')
  const tsGoCollections = path.join(sourceDir, 'typescript-go', 'internal', 'collections')
  if (!fs.existsSync(collectionsDir) && fs.existsSync(tsGoCollections)) {
    fs.mkdirSync(collectionsDir, { recursive: true })
    const files = fs.readdirSync(tsGoCollections).filter((f) => {
      return f.endsWith('.go') && !f.endsWith('_test.go')
    })
    for (const file of files) {
      fs.copyFileSync(path.join(tsGoCollections, file), path.join(collectionsDir, file))
    }
    console.log(`Copied ${files.length} collection files`)
  }

  // write ready marker
  fs.writeFileSync(readyMarker, new Date().toISOString())
  console.log('tsgolint source ready')

  return sourceDir
}

export function cachedBinaryExists(contentHash: string): boolean {
  const binPath = getBinaryPath(contentHash)
  try {
    fs.accessSync(binPath, fs.constants.X_OK)
    return true
  } catch {
    return false
  }
}
