// lintcn lint — build a custom tsgolint binary and run it against the project.
// Handles Go workspace generation, compilation with caching, and execution.

import fs from 'node:fs'
import { spawn } from 'node:child_process'
import { requireLintcnDir } from '../paths.ts'
import { discoverRules } from '../discover.ts'
import { generateBuildWorkspace } from '../codegen.ts'
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
  const contentHash = await computeContentHash({
    lintcnDir,
    tsgolintVersion,
  })

  // check cache
  if (!rebuild && cachedBinaryExists(contentHash)) {
    console.log('Using cached binary')
    return getBinaryPath(contentHash)
  }

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

  const { exitCode: buildExitCode } = await execAsync('go', ['build', '-o', binaryPath, './wrapper'], {
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
}: {
  rebuild: boolean
  tsgolintVersion: string
  passthroughArgs: string[]
}): Promise<number> {
  const binaryPath = await buildBinary({ rebuild, tsgolintVersion })

  // run the binary with passthrough args, inheriting stdio
  return new Promise((resolve) => {
    const proc = spawn(binaryPath, passthroughArgs, {
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
