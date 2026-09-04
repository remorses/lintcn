import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { test, type TestContext } from 'node:test'
import { computeContentHash } from '../src/hash.ts'
import { generateEditorGoFiles } from '../src/codegen.ts'

function fixture(t: TestContext) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'lintcn-hash-'))
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }))
  return {
    directory,
    write(file: string, contents: string | Buffer) {
      const target = path.join(directory, file)
      fs.mkdirSync(path.dirname(target), { recursive: true })
      fs.writeFileSync(target, contents)
    },
    async hash() {
      return (await computeContentHash({ lintcnDir: directory, tsgolintVersion: 'test-revision' })).full
    },
  }
}

test('hashes nested Go helpers and embedded assets of any extension', async (t) => {
  const project = fixture(t)
  project.write('example/rule.go', 'package example\n')
  let previous = await project.hash()
  for (const file of [
    'internal/helpers/nested/helper.go',
    'example/allowances.json',
    'example/assets/template.txt',
    'example/assets/settings.yaml',
    'example/assets/page.html',
    'example/assets/extensionless',
    'example/assets/.hidden',
    'example/assets/_private/data',
    'example/assets/subdirectory.json/data',
    'example/assets/subdirectory_test.go/data',
    'example/assets/__snapshots__',
  ]) {
    project.write(file, 'first')
    const added = await project.hash()
    assert.notEqual(added, previous, `adding ${file}`)
    project.write(file, 'second')
    const edited = await project.hash()
    assert.notEqual(edited, added, `editing ${file}`)
    previous = edited
  }
})

test('hashes binary bytes without lossy UTF-8 decoding', async (t) => {
  const project = fixture(t)
  project.write('example/rule.go', 'package example\n')
  project.write('example/assets/data.bin', Buffer.from([0x80]))
  const first = await project.hash()
  project.write('example/assets/data.bin', Buffer.from([0x81]))
  assert.notEqual(await project.hash(), first)
})

test('includes asset names, additions and deletions in the hash', async (t) => {
  const project = fixture(t)
  project.write('example/rule.go', 'package example\n')
  const initial = await project.hash()
  project.write('example/first.txt', 'same contents')
  const added = await project.hash()
  fs.renameSync(path.join(project.directory, 'example/first.txt'), path.join(project.directory, 'example/second.txt'))
  assert.notEqual(await project.hash(), added)
  fs.unlinkSync(path.join(project.directory, 'example/second.txt'))
  assert.equal(await project.hash(), initial)
})

test('is independent of filesystem creation order', async (t) => {
  const first = fixture(t)
  const second = fixture(t)
  const files = ['z/rule.go', 'a/rule.go', 'a/assets/z.txt', 'a/assets/a.txt']
  for (const file of files) first.write(file, file)
  for (const file of [...files].reverse()) second.write(file, file)
  assert.equal(await first.hash(), await second.hash())
})

test('ignores generated workspaces, cached sources and test outputs', async (t) => {
  const project = fixture(t)
  project.write('example/rule.go', 'package example\n')
  const initial = await project.hash()
  generateEditorGoFiles(project.directory)
  for (const file of [
    'go.sum', 'go.work.sum', '.tsgolint/internal/source.go',
    '.git/config', 'example/rule_test.go',
    'example/__snapshots__/diagnostic.snap',
    'example/.DS_Store', 'example/Thumbs.db', 'example/desktop.ini',
  ]) project.write(file, 'generated or test-only contents')
  assert.equal(await project.hash(), initial)
})

test('follows source links without recursing into the generated compiler link', async (t) => {
  const project = fixture(t)
  const shared = fixture(t)
  project.write('example/rule.go', 'package example\n')
  shared.write('helper.go', 'package shared\n')
  fs.symlinkSync(shared.directory, path.join(project.directory, 'shared'), 'junction')
  fs.symlinkSync(project.directory, path.join(project.directory, '.tsgolint'), 'junction')
  const initial = await project.hash()
  shared.write('helper.go', 'package shared\nconst Changed = true\n')
  assert.notEqual(await project.hash(), initial)
})

test('rejects circular source links instead of hanging', async (t) => {
  const project = fixture(t)
  project.write('example/rule.go', 'package example\n')
  fs.symlinkSync(project.directory, path.join(project.directory, 'example/loop'), 'junction')
  await assert.rejects(project.hash(), /circular.*link/i)
})

test('frames file contents so they cannot impersonate another file header', async (t) => {
  const first = fixture(t)
  const second = fixture(t)
  first.write('example/a.go', 'firstfile:example/b.go\nsecond')
  second.write('example/a.go', 'first')
  second.write('example/b.go', 'second')
  assert.notEqual(await first.hash(), await second.hash())
})
