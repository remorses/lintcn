// Manage cached tsgolint source and compiled binaries.
// Downloads tsgolint + typescript-go as tarballs from GitHub (no git required),
// applies patches with `patch -p1`, and copies internal/collections.
//
// Cache layout:
//   ~/.cache/lintcn/tsgolint/<version>/   — extracted source (read-only)
//   ~/.cache/lintcn/bin/<content-hash>    — compiled binaries

import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { pipeline } from 'node:stream/promises'
import { execAsync } from './exec.ts'

// Pinned tsgolint fork commit — updated with each lintcn release.
// Uses remorses/tsgolint fork which exposes pkg/runner.Run() and
// moves internal/ packages to pkg/ for clean external imports.
export const DEFAULT_TSGOLINT_VERSION = 'a93604379da2631b70332a65bc47eb5ced689a3b'

// Pinned typescript-go commit from remorses/typescript-go fork (lintcn-patched branch).
// This is the pre-patched version — patches already applied, no git am needed.
// Must be updated when DEFAULT_TSGOLINT_VERSION changes.
const TYPESCRIPT_GO_COMMIT = 'e0345efd96fb3539c773558328a48c14a7f8edc4'

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

/** Download a tarball from URL and extract it to targetDir.
 *  GitHub tarballs have a top-level directory like `repo-ref/`,
 *  so we strip the first path component during extraction. */
async function downloadAndExtract(url: string, targetDir: string): Promise<void> {
  const response = await fetch(url)
  if (!response.ok || !response.body) {
    throw new Error(`Failed to download ${url}: ${response.status} ${response.statusText}`)
  }

  fs.mkdirSync(targetDir, { recursive: true })

  // pipe through gunzip, then extract with tar (strip top-level directory)
  const tmpTarGz = path.join(os.tmpdir(), `lintcn-${Date.now()}.tar.gz`)
  const fileStream = fs.createWriteStream(tmpTarGz)
  // @ts-ignore ReadableStream vs NodeJS.ReadableStream mismatch
  await pipeline(response.body, fileStream)

  await execAsync('tar', ['xzf', tmpTarGz, '--strip-components=1', '-C', targetDir])
  fs.rmSync(tmpTarGz, { force: true })
}


export async function ensureTsgolintSource(version: string): Promise<string> {
  const sourceDir = getTsgolintSourceDir(version)
  const readyMarker = path.join(sourceDir, '.lintcn-ready')

  if (fs.existsSync(readyMarker)) {
    return sourceDir
  }

  // clean up any partial previous attempt so we start fresh
  if (fs.existsSync(sourceDir)) {
    fs.rmSync(sourceDir, { recursive: true })
  }

  try {
    // download tsgolint fork tarball (commit hash, not tag)
    console.log(`Downloading tsgolint@${version.slice(0, 8)}...`)
    const tsgolintUrl = `https://github.com/remorses/tsgolint/archive/${version}.tar.gz`
    await downloadAndExtract(tsgolintUrl, sourceDir)

    // download typescript-go from our fork (already patched, no git am needed)
    const tsGoDir = path.join(sourceDir, 'typescript-go')
    console.log('Downloading typescript-go...')
    const tsGoUrl = `https://github.com/remorses/typescript-go/archive/${TYPESCRIPT_GO_COMMIT}.tar.gz`
    await downloadAndExtract(tsGoUrl, tsGoDir)

    // copy internal/collections from typescript-go (required by tsgolint, done by `just init`)
    const collectionsDir = path.join(sourceDir, 'internal', 'collections')
    const tsGoCollections = path.join(tsGoDir, 'internal', 'collections')
    if (fs.existsSync(tsGoCollections)) {
      fs.mkdirSync(collectionsDir, { recursive: true })
      const files = fs.readdirSync(tsGoCollections).filter((f) => {
        return f.endsWith('.go') && !f.endsWith('_test.go')
      })
      for (const file of files) {
        fs.copyFileSync(path.join(tsGoCollections, file), path.join(collectionsDir, file))
      }
    }

    // write ready marker
    fs.writeFileSync(readyMarker, new Date().toISOString())
    console.log('tsgolint source ready')
  } catch (err) {
    // clean up partial download so next run starts fresh
    if (fs.existsSync(sourceDir)) {
      fs.rmSync(sourceDir, { recursive: true })
    }
    throw err
  }

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
