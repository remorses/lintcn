// Scan .lintcn/*/ subfolders for rule.Rule variables and lintcn: metadata comments.
// Each subfolder is a Go package containing one rule with its companions.
// Returns structured info about each discovered rule for codegen and list display.

import fs from 'node:fs'
import path from 'node:path'

export interface RuleMetadata {
  /** kebab-case rule name from // lintcn:name or derived from folder name */
  name: string
  /** one-line description from // lintcn:description */
  description: string
  /** original source URL from // lintcn:source */
  source: string
  /** exported Go variable name like NoFloatingPromisesRule */
  varName: string
  /** Go package name (= subfolder name, e.g. no_floating_promises) */
  packageName: string
}

// Matches `var XxxRule = rule.Rule{` with optional leading whitespace
// and optional import alias (e.g. `r.Rule{` if imported as `r "...rule"`)
const RULE_VAR_RE = /^\s*var\s+(\w+)\s*=\s*\w*\.?Rule\s*\{/m
const METADATA_RE = /^\/\/\s*lintcn:(\w+)\s+(.+)$/gm

export function parseMetadata(content: string): Record<string, string> {
  const meta: Record<string, string> = {}
  for (const match of content.matchAll(METADATA_RE)) {
    meta[match[1]] = match[2].trim()
  }
  return meta
}

export function parseRuleVar(content: string): string | undefined {
  const match = content.match(RULE_VAR_RE)
  return match?.[1]
}

export function discoverRules(lintcnDir: string): RuleMetadata[] {
  if (!fs.existsSync(lintcnDir)) {
    return []
  }

  const rules: RuleMetadata[] = []
  const entries = fs.readdirSync(lintcnDir, { withFileTypes: true })

  // Warn about flat .go files that should be in subfolders
  const flatGoFiles = entries.filter((e) => {
    return e.isFile() && e.name.endsWith('.go')
  })
  for (const file of flatGoFiles) {
    const baseName = file.name.replace(/(_test)?\.go$/, '')
    console.error(
      `Error: ${file.name} is a flat file in .lintcn/ — rules must be in subfolders.\n` +
      `  Move it to .lintcn/${baseName}/${file.name} and change the package to "${baseName}".\n`,
    )
  }

  for (const entry of entries) {
    if (!entry.isDirectory() || entry.name.startsWith('.')) continue

    const subDir = path.join(lintcnDir, entry.name)
    const goFiles = fs.readdirSync(subDir).filter((f) => {
      return f.endsWith('.go') && !f.endsWith('_test.go')
    })

    for (const fileName of goFiles) {
      const filePath = path.join(subDir, fileName)
      const content = fs.readFileSync(filePath, 'utf-8')

      const varName = parseRuleVar(content)
      if (!varName) continue

      const meta = parseMetadata(content)

      rules.push({
        name: meta.name || entry.name.replace(/_/g, '-'),
        description: meta.description || '',
        source: meta.source || '',
        varName,
        packageName: entry.name,
      })
    }
  }

  return rules
}
