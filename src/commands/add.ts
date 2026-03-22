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
  ref: string
  /** Path to the directory containing the rule files */
  dirPath: string
  /** Set when URL points to a specific file (not a folder) */
  fileName?: string
}

/** Parse GitHub blob/tree/raw URLs into components.
 *  Ref is assumed to be the first path component after blob/tree —
 *  branch names with slashes (e.g. feature/foo) are not supported. */
function parseGitHubUrl(url: string): ParsedGitHubUrl | null {
  // GitHub blob URLs: github.com/owner/repo/blob/ref/path/to/file.go
  let match = url.match(/^https?:\/\/github\.com\/([^/]+)\/([^/]+)\/blob\/([^/]+)\/(.+)$/)
  if (match) {
    const [, owner, repo, ref, filePath] = match
    return { owner, repo, ref, dirPath: path.posix.dirname(filePath), fileName: path.posix.basename(filePath) }
  }

  // GitHub tree URLs: github.com/owner/repo/tree/ref/path/to/folder
  match = url.match(/^https?:\/\/github\.com\/([^/]+)\/([^/]+)\/tree\/([^/]+)\/(.+)$/)
  if (match) {
    const [, owner, repo, ref, dirPath] = match
    return { owner, repo, ref, dirPath }
  }

  // raw.githubusercontent.com URLs
  match = url.match(/^https?:\/\/raw\.githubusercontent\.com\/([^/]+)\/([^/]+)\/([^/]+)\/(.+)$/)
  if (match) {
    const [, owner, repo, ref, filePath] = match
    return { owner, repo, ref, dirPath: path.posix.dirname(filePath), fileName: path.posix.basename(filePath) }
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

export async function addRule(url: string): Promise<void> {
  const parsed = parseGitHubUrl(url)
  if (!parsed) {
    throw new Error(
      'Only GitHub URLs are supported. Pass a /blob/ (file) or /tree/ (folder) URL.\n' +
      'Example: lintcn add https://github.com/oxc-project/tsgolint/tree/main/internal/rules/no_floating_promises',
    )
  }

  const { owner, repo, ref, dirPath } = parsed
  const folderName = path.posix.basename(dirPath)

  console.log(`Fetching ${owner}/${repo}/${dirPath}...`)
  const items = await listGitHubFolder(owner, repo, dirPath, ref)

  // Filter for .go and .json files
  const filesToFetch = items.filter((item) => {
    return item.type === 'file' && item.download_url && (item.name.endsWith('.go') || item.name.endsWith('.json'))
  })

  if (filesToFetch.length === 0) {
    throw new Error(`No .go files found in ${dirPath}. Is this a rule folder?`)
  }

  // Warn if this doesn't look like a single-rule folder (too many main .go files)
  const mainGoFiles = filesToFetch.filter((f) => {
    return f.name.endsWith('.go') && !f.name.endsWith('_test.go') && f.name !== 'options.go'
  })
  if (mainGoFiles.length > 3) {
    console.warn(
      `Warning: folder has ${mainGoFiles.length} non-test .go files. ` +
      `This may be a directory of multiple rules — consider using a more specific URL.`,
    )
  }

  const lintcnDir = getLintcnDir()
  const ruleDir = path.join(lintcnDir, folderName)

  // Clean existing rule folder if it exists
  if (fs.existsSync(ruleDir)) {
    fs.rmSync(ruleDir, { recursive: true })
    console.log(`Overwriting existing ${folderName}/`)
  }

  fs.mkdirSync(ruleDir, { recursive: true })

  // Fetch and write all files
  for (const item of filesToFetch) {
    let content = await fetchFile(item.download_url!)

    // Add lintcn:source comment to the main rule file (same name as folder)
    if (item.name === `${folderName}.go`) {
      content = ensureSourceComment(content, url)
    }

    fs.writeFileSync(path.join(ruleDir, item.name), content)
    console.log(`  ${item.name}`)
  }

  console.log(`Added ${folderName}/ (${filesToFetch.length} files)`)

  // Ensure tsgolint source is available
  const tsgolintDir = await ensureTsgolintSource(DEFAULT_TSGOLINT_VERSION)

  // Create/refresh .tsgolint symlink for gopls
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
