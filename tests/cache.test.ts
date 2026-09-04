import assert from 'node:assert/strict'
import { execFile } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { test, type TestContext } from 'node:test'
import { promisify } from 'node:util'

const execFileAsync = promisify(execFile)

async function runIsolated(t: TestContext, body: string) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'lintcn-cache-'))
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }))
  const rules = path.join(directory, '.lintcn/example')
  fs.mkdirSync(rules, { recursive: true })
  fs.writeFileSync(path.join(rules, 'rule.go'), 'package example\nvar ExampleRule = rule.Rule{Name: "example"}\n')
  const script = `
    import assert from 'node:assert/strict'
    import fs from 'node:fs'
    import os from 'node:os'
    import path from 'node:path'
    import { mock } from 'node:test'
    import { buildBinary } from ${JSON.stringify(new URL('../src/commands/lint.ts', import.meta.url).href)}
    import { computeContentHash } from ${JSON.stringify(new URL('../src/hash.ts', import.meta.url).href)}
    import { getBinDir, getBinaryPath, getBuildLockDir, getTsgolintSourceDir, acquireCacheLock } from ${JSON.stringify(new URL('../src/cache.ts', import.meta.url).href)}
    mock.method(os, 'homedir', () => ${JSON.stringify(directory)})
    mock.method(globalThis, 'fetch', () => { throw new Error('Unexpected network request') })
    const version = 'test-revision'
    const lintcnDir = path.join(process.cwd(), '.lintcn')
    const { short: hash } = await computeContentHash({ lintcnDir, tsgolintVersion: version })
    const binary = getBinaryPath(hash)
    fs.mkdirSync(getBinDir(), { recursive: true })
    const writeBinary = () => fs.writeFileSync(binary, 'cached binary fixture', { mode: 0o755 })
    ${body}
  `
  const result = await execFileAsync(process.execPath, [
    '--import', import.meta.resolve('tsx'), '--input-type=module', '--eval', script,
  ], {
    cwd: directory,
    env: { ...process.env, PATH: path.join(directory, 'no-executables') },
    timeout: 10_000,
  })
  assert.equal(result.stderr, '')
}

test('returns a cached binary without Go, downloads or generated workspaces', async (t) => {
  await runIsolated(t, `
    writeBinary()
    assert.equal(await buildBinary({ rebuild: false, tsgolintVersion: version }), binary)
    assert.equal(fs.existsSync(getTsgolintSourceDir(version)), false)
    assert.equal(fs.existsSync(path.join(lintcnDir, 'go.mod')), false)
  `)
})

test('rechecks the cache after acquiring the build lock before requiring Go', async (t) => {
  await runIsolated(t, `
    const lock = getBuildLockDir(hash)
    const mkdir = fs.mkdirSync.bind(fs)
    mock.method(fs, 'mkdirSync', (directory, options) => {
      const result = mkdir(directory, options)
      if (directory === lock) writeBinary()
      return result
    })
    assert.equal(await buildBinary({ rebuild: false, tsgolintVersion: version }), binary)
    assert.equal(fs.existsSync(lock), false)
  `)
})

test('still requires Go on misses and forced rebuilds, releasing the build lock on failure', async (t) => {
  await runIsolated(t, `
    await assert.rejects(buildBinary({ rebuild: false, tsgolintVersion: version }), /Go is required/)
    assert.equal(fs.existsSync(getBuildLockDir(hash)), false)
    writeBinary()
    await assert.rejects(buildBinary({ rebuild: true, tsgolintVersion: version }), /Go is required/)
    assert.equal(fs.existsSync(getBuildLockDir(hash)), false)
  `)
})

test('retries existing cache locks and propagates other filesystem failures', async (t) => {
  await runIsolated(t, `
    const lock = getBuildLockDir(hash)
    fs.mkdirSync(lock, { recursive: true })
    setTimeout(() => fs.rmSync(lock, { recursive: true }), 25)
    assert.equal(await acquireCacheLock(lock, 'test lock'), lock)
    const failure = Object.assign(new Error('access denied'), { code: 'EACCES' })
    const mkdir = fs.mkdirSync.bind(fs)
    mock.method(fs, 'mkdirSync', (directory, options) => {
      if (directory === lock) throw failure
      return mkdir(directory, options)
    })
    await assert.rejects(acquireCacheLock(lock, 'test lock'), (error) => error === failure)
  `)
})
