// Manage cached tsgolint source and compiled binaries.
// Downloads tsgolint fork + typescript-go as tarballs from GitHub,
// applies tsgolint's patches to typescript-go, and copies collections.
//
// Cache layout:
//   ~/.cache/lintcn/tsgolint/<version>/   — extracted source (read-only)
//   ~/.cache/lintcn/build/<content-hash>/ — per-hash build workspace (no race)
//   ~/.cache/lintcn/bin/<content-hash>    — compiled binaries

import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import crypto from 'node:crypto'
import { pipeline } from 'node:stream/promises'
import { extract } from 'tar'
import { execAsync } from './exec.ts'

// Pinned tsgolint fork commit — updated with each lintcn release.
// Uses remorses/tsgolint fork which adds internal/runner.Run() and
// TSGOLINT_SNAPSHOT_CWD env var for cwd-relative test snapshots.
export const DEFAULT_TSGOLINT_VERSION = '4e4666c284d3b5cf7fa082523a369ef507e4360c'

// Pinned typescript-go base commit from microsoft/typescript-go (before patches).
// Patches from tsgolint/patches/ are applied on top during setup.
// Must be updated when DEFAULT_TSGOLINT_VERSION changes.
const TYPESCRIPT_GO_COMMIT = 'c0703e66b68b826eedadce353d63fe9f4ea21fb6'

// Strict pattern for version strings — prevents path traversal via ../
const VERSION_PATTERN = /^[a-zA-Z0-9._-]+$/

/** Validate version string to prevent path traversal attacks.
 *  Only allows alphanumeric chars, dots, underscores, and hyphens. */
export function validateVersion(version: string): void {
  if (!VERSION_PATTERN.test(version)) {
    throw new Error(
      `Invalid tsgolint version "${version}". ` +
      'Version must only contain alphanumeric characters, dots, underscores, and hyphens.',
    )
  }
}

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

/** Per-hash build directory to avoid races between concurrent lintcn processes. */
export function getBuildDir(contentHash: string): string {
  return path.join(getCacheDir(), 'build', contentHash)
}

/** Download a tarball from URL and extract it to targetDir.
 *  Uses the `tar` npm package for cross-platform support (no shell tar needed).
 *  GitHub tarballs have a top-level directory like `repo-ref/`,
 *  so we strip the first path component during extraction. */
async function downloadAndExtract(url: string, targetDir: string): Promise<void> {
  const controller = new AbortController()
  const timeout = setTimeout(() => {
    controller.abort(new Error(`Download timed out after 120s: ${url}`))
  }, 120_000)

  let response: Response
  try {
    response = await fetch(url, { signal: controller.signal })
  } finally {
    clearTimeout(timeout)
  }

  if (!response.ok || !response.body) {
    throw new Error(`Failed to download ${url}: ${response.status} ${response.statusText}`)
  }

  // download to temp file with random suffix to avoid collisions
  const tmpTarGz = path.join(os.tmpdir(), `lintcn-${crypto.randomBytes(8).toString('hex')}.tar.gz`)
  try {
    const fileStream = fs.createWriteStream(tmpTarGz)
    // @ts-ignore ReadableStream vs NodeJS.ReadableStream mismatch
    await pipeline(response.body, fileStream)

    // extract with npm tar package (cross-platform, no shell tar needed)
    fs.mkdirSync(targetDir, { recursive: true })
    await extract({
      file: tmpTarGz,
      cwd: targetDir,
      strip: 1,
    })
  } finally {
    // always clean up temp file
    fs.rmSync(tmpTarGz, { force: true })
  }
}

/** Apply git-format patches using `patch -p1` (no git required).
 *  Patches are standard unified diff format, `patch` ignores the git metadata. */
async function applyPatches(patchesDir: string, targetDir: string): Promise<number> {
  const patches = fs.readdirSync(patchesDir)
    .filter((f) => { return f.endsWith('.patch') })
    .sort()

  for (const patchFile of patches) {
    const patchPath = path.join(patchesDir, patchFile)
    await execAsync('patch', ['-p1', '--batch', '-i', patchPath], { cwd: targetDir })
  }

  return patches.length
}

export async function ensureTsgolintSource(version: string): Promise<string> {
  validateVersion(version)

  const sourceDir = getTsgolintSourceDir(version)
  const readyMarker = path.join(sourceDir, '.lintcn-ready')

  if (fs.existsSync(readyMarker)) {
    return sourceDir
  }

  // Use a temp directory for the download, then atomic rename on success.
  // This prevents concurrent processes from seeing partial state, and
  // avoids the "non-empty dir on retry" problem.
  const tmpDir = path.join(getCacheDir(), 'tsgolint', `.tmp-${version}-${crypto.randomBytes(4).toString('hex')}`)

  // clean up any partial previous attempt
  if (fs.existsSync(sourceDir)) {
    fs.rmSync(sourceDir, { recursive: true })
  }

  try {
    // download tsgolint fork tarball
    console.log(`Downloading tsgolint@${version.slice(0, 8)}...`)
    const tsgolintUrl = `https://github.com/remorses/tsgolint/archive/${version}.tar.gz`
    await downloadAndExtract(tsgolintUrl, tmpDir)

    // download typescript-go from microsoft (base commit before patches)
    const tsGoDir = path.join(tmpDir, 'typescript-go')
    console.log('Downloading typescript-go...')
    const tsGoUrl = `https://github.com/microsoft/typescript-go/archive/${TYPESCRIPT_GO_COMMIT}.tar.gz`
    await downloadAndExtract(tsGoUrl, tsGoDir)

    // apply tsgolint's patches to typescript-go
    const patchesDir = path.join(tmpDir, 'patches')
    if (fs.existsSync(patchesDir)) {
      const count = await applyPatches(patchesDir, tsGoDir)
      if (count > 0) {
        console.log(`Applied ${count} patches`)
      }
    }

    // copy internal/collections from typescript-go (required by tsgolint, done by `just init`)
    const collectionsDir = path.join(tmpDir, 'internal', 'collections')
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
    fs.writeFileSync(path.join(tmpDir, '.lintcn-ready'), new Date().toISOString())

    // atomic rename: move completed dir to final location
    fs.mkdirSync(path.dirname(sourceDir), { recursive: true })
    fs.renameSync(tmpDir, sourceDir)

    console.log('tsgolint source ready')
  } catch (err) {
    // clean up partial temp directory
    if (fs.existsSync(tmpDir)) {
      fs.rmSync(tmpDir, { recursive: true })
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
