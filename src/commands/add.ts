// lintcn add <url> — fetch a rule folder by URL and copy into .lintcn/{rule_name}/
// Supports GitHub folder URLs (/tree/) and file URLs (/blob/).
// For file URLs, auto-detects the parent folder and fetches all sibling files.
// Uses GitHub API to list folder contents.

import fs from 'node:fs'
import path from 'node:path'
import { execSync } from 'node:child_process'
import { getLintcnDir } from '../paths.ts'
import { generateEditorGoFiles } from '../codegen.ts'
import { ensureTsgolintSource, DEFAULT_TSGOLINT_VERSION } from '../cache.ts'

interface ParsedGitHubUrl {
  owner: string
  repo: string
  /** Branch/tag/commit. Undefined for bare repo URLs — resolve via API. */
  ref: string | undefined
  /** Path to the directory containing the rule files. Empty string for repo root. */
  dirPath: string
  /** Set when URL points to a specific file (not a folder) */
  fileName?: string
}

/** Parse any GitHub URL into components.
 *  Supports: bare repo, /tree/ folders, /blob/ files, raw.githubusercontent.com.
 *  Ref is the first path component after blob/tree — branch names with slashes
 *  (e.g. feature/foo) are not supported. */
function parseGitHubUrl(url: string): ParsedGitHubUrl | null {
  let hostname: string
  let segments: string[]
  try {
    const u = new URL(url)
    hostname = u.hostname
    segments = u.pathname.replace(/\/$/, '').split('/').filter(Boolean)
  } catch {
    return null
  }

  // raw.githubusercontent.com/owner/repo/ref/path/to/file
  if (hostname === 'raw.githubusercontent.com') {
    if (segments.length < 4) return null
    const [owner, repo, ref, ...rest] = segments
    const filePath = rest.join('/')
    return { owner, repo, ref, dirPath: path.posix.dirname(filePath), fileName: path.posix.basename(filePath) }
  }

  if (hostname !== 'github.com') return null
  if (segments.length < 2) return null

  const [owner, repo, kind, ref, ...rest] = segments

  // Bare repo URL: github.com/owner/repo
  if (!kind) {
    return { owner, repo, ref: undefined, dirPath: '' }
  }

  const subPath = rest.join('/')

  if (kind === 'tree') {
    if (!ref) return null
    return { owner, repo, ref, dirPath: subPath }
  }

  if (kind === 'blob') {
    if (!ref || !subPath) return null
    return { owner, repo, ref, dirPath: path.posix.dirname(subPath), fileName: path.posix.basename(subPath) }
  }

  return null
}

interface GitHubContentItem {
  name: string
  download_url: string | null
  type: 'file' | 'dir'
}

/** Get a GitHub auth token from gh CLI, GITHUB_TOKEN env, or return undefined. */
function getGitHubToken(): string | undefined {
  if (process.env.GITHUB_TOKEN) {
    return process.env.GITHUB_TOKEN
  }
  // Try gh CLI token (synchronous to keep it simple)
  try {
    return execSync('gh auth token', { encoding: 'utf-8', stdio: ['pipe', 'pipe', 'pipe'] }).trim() || undefined
  } catch {
    return undefined
  }
}

/** Resolve the default branch for a repo (e.g. "main", "master"). */
async function resolveDefaultBranch(owner: string, repo: string): Promise<string> {
  const apiUrl = `https://api.github.com/repos/${owner}/${repo}`
  const headers: Record<string, string> = {
    'Accept': 'application/vnd.github.v3+json',
    'User-Agent': 'lintcn',
  }
  const token = getGitHubToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const response = await fetch(apiUrl, { headers })
  if (!response.ok) {
    throw new Error(`GitHub API error: ${response.status} ${response.statusText}\n  ${apiUrl}`)
  }
  const data = (await response.json()) as { default_branch: string }
  return data.default_branch
}

/** Files/dirs in .lintcn/ that are generated and should not be treated as rule folders. */
const LINTCN_GENERATED = new Set([
  '.tsgolint', '.gitignore', 'go.work', 'go.work.sum', 'go.mod', 'go.sum',
])

/** List files in a GitHub directory via the Contents API. */
async function listGitHubFolder(owner: string, repo: string, dirPath: string, ref: string): Promise<GitHubContentItem[]> {
  const apiUrl = `https://api.github.com/repos/${owner}/${repo}/contents/${dirPath}?ref=${ref}`
  const headers: Record<string, string> = {
    'Accept': 'application/vnd.github.v3+json',
    'User-Agent': 'lintcn',
  }
  const token = getGitHubToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const response = await fetch(apiUrl, { headers })

  if (!response.ok) {
    throw new Error(`GitHub API error: ${response.status} ${response.statusText}\n  ${apiUrl}`)
  }

  const data = await response.json()
  if (!Array.isArray(data)) {
    throw new Error(`Expected a directory listing from GitHub API but got a single file.\n  ${apiUrl}`)
  }

  return data as GitHubContentItem[]
}

async function fetchFile(url: string): Promise<string> {
  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(`Failed to fetch ${url}: ${response.status} ${response.statusText}`)
  }
  return response.text()
}

function ensureSourceComment(content: string, sourceUrl: string): string {
  if (content.includes('// lintcn:source')) {
    return content
  }
  // Insert source comment after any existing lintcn: comment block, or at the very top
  const lines = content.split('\n')
  let insertIndex = 0
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].startsWith('// lintcn:')) {
      insertIndex = i + 1
    } else if (insertIndex > 0) {
      break
    }
  }
  lines.splice(insertIndex, 0, `// lintcn:source ${sourceUrl}`)
  return lines.join('\n')
}

/** Download a single rule folder into .lintcn/{folderName}/.
 *  Overwrites existing folder if present. */
async function downloadSingleRule(
  owner: string, repo: string, ref: string, dirPath: string,
  lintcnDir: string, sourceUrl: string,
): Promise<void> {
  const folderName = path.posix.basename(dirPath)
  const items = await listGitHubFolder(owner, repo, dirPath, ref)

  const filesToFetch = items.filter((item) => {
    return item.type === 'file' && item.download_url && (item.name.endsWith('.go') || item.name.endsWith('.json'))
  })

  if (filesToFetch.length === 0) {
    console.warn(`  Skipping ${folderName}/ — no .go files found`)
    return
  }

  const ruleDir = path.join(lintcnDir, folderName)

  if (fs.existsSync(ruleDir)) {
    fs.rmSync(ruleDir, { recursive: true })
    console.log(`  Overwriting existing ${folderName}/`)
  }

  fs.mkdirSync(ruleDir, { recursive: true })

  for (const item of filesToFetch) {
    let content = await fetchFile(item.download_url!)

    if (item.name === `${folderName}.go`) {
      content = ensureSourceComment(content, sourceUrl)
    }

    fs.writeFileSync(path.join(ruleDir, item.name), content)
    console.log(`    ${item.name}`)
  }

  console.log(`  Added ${folderName}/ (${filesToFetch.length} files)`)
}

/** Ensure tsgolint source, refresh symlink, regenerate go.work/go.mod. */
async function finalizeLintcnDir(lintcnDir: string): Promise<void> {
  const tsgolintDir = await ensureTsgolintSource(DEFAULT_TSGOLINT_VERSION)

  const tsgolintLink = path.join(lintcnDir, '.tsgolint')
  try {
    fs.lstatSync(tsgolintLink)
    fs.rmSync(tsgolintLink, { force: true })
  } catch {
    // doesn't exist
  }
  fs.symlinkSync(tsgolintDir, tsgolintLink)

  generateEditorGoFiles(lintcnDir)
  console.log('Editor support files generated (go.work, go.mod)')
}

/** Download all rule subfolders from a remote .lintcn/ directory.
 *  Each subfolder is treated as a separate rule. Local rules not present
 *  in the remote are preserved (merge, not replace). */
async function addLintcnFolder(
  owner: string, repo: string, ref: string, lintcnPath: string, sourceUrl: string,
): Promise<void> {
  console.log(`Fetching .lintcn/ from ${owner}/${repo}...`)
  const items = await listGitHubFolder(owner, repo, lintcnPath, ref)

  const ruleDirs = items.filter((item) => {
    return item.type === 'dir' && !LINTCN_GENERATED.has(item.name) && !item.name.startsWith('.')
  })

  if (ruleDirs.length === 0) {
    throw new Error(`No rule folders found in ${lintcnPath}. Is this a .lintcn/ directory?`)
  }

  console.log(`Found ${ruleDirs.length} rule(s)`)

  const lintcnDir = getLintcnDir()

  for (const dir of ruleDirs) {
    const ruleDirPath = lintcnPath ? `${lintcnPath}/${dir.name}` : dir.name
    await downloadSingleRule(owner, repo, ref, ruleDirPath, lintcnDir, sourceUrl)
  }

  await finalizeLintcnDir(lintcnDir)
  console.log(`\nDone — added ${ruleDirs.length} rule(s) from ${owner}/${repo}`)
}

export async function addRule(url: string): Promise<void> {
  const parsed = parseGitHubUrl(url)
  if (!parsed) {
    throw new Error(
      'Only GitHub URLs are supported.\n' +
      'Examples:\n' +
      '  lintcn add https://github.com/someone/their-project\n' +
      '  lintcn add https://github.com/oxc-project/tsgolint/tree/main/internal/rules/no_floating_promises',
    )
  }

  const { owner, repo, fileName } = parsed
  let { ref, dirPath } = parsed

  // Bare repo URL — resolve default branch and look for .lintcn/ at root
  if (ref === undefined) {
    ref = await resolveDefaultBranch(owner, repo)
    console.log(`Resolved default branch: ${ref}`)

    const rootItems = await listGitHubFolder(owner, repo, '', ref)
    const lintcnEntry = rootItems.find((item) => item.type === 'dir' && item.name === '.lintcn')
    if (!lintcnEntry) {
      throw new Error(
        `No .lintcn/ directory found in ${owner}/${repo}.\n` +
        'The repo needs a .lintcn/ folder with rule subfolders.',
      )
    }

    await addLintcnFolder(owner, repo, ref, '.lintcn', url)
    return
  }

  // For blob/raw URLs, dirPath already points at the parent folder
  // For tree URLs, dirPath is the folder itself — but it might be a .lintcn/ collection
  if (!fileName) {
    // Tree URL — check if this is a collection (has subdirectories) or a single rule
    const items = await listGitHubFolder(owner, repo, dirPath, ref)
    const subdirs = items.filter((item) => {
      return item.type === 'dir' && !LINTCN_GENERATED.has(item.name) && !item.name.startsWith('.')
    })

    if (subdirs.length > 0) {
      // Looks like a collection of rules (e.g. .lintcn/ or rules/)
      await addLintcnFolder(owner, repo, ref, dirPath, url)
      return
    }
  }

  // Single rule — download one folder
  const folderName = path.posix.basename(dirPath)

  console.log(`Fetching ${owner}/${repo}/${dirPath}...`)
  const lintcnDir = getLintcnDir()

  // Warn if this doesn't look like a single-rule folder (too many main .go files)
  const items = await listGitHubFolder(owner, repo, dirPath, ref)
  const mainGoFiles = items.filter((f) => {
    return f.type === 'file' && f.name.endsWith('.go') && !f.name.endsWith('_test.go') && f.name !== 'options.go'
  })
  if (mainGoFiles.length > 3) {
    console.warn(
      `Warning: folder has ${mainGoFiles.length} non-test .go files. ` +
      `This may be a directory of multiple rules — consider using a more specific URL.`,
    )
  }

  await downloadSingleRule(owner, repo, ref, dirPath, lintcnDir, url)
  await finalizeLintcnDir(lintcnDir)
}
