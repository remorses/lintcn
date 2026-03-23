// Scan .lintcn/*/ subfolders for rule.Rule variables and lintcn: metadata comments.
// Each subfolder is a Go package containing one rule with its companions.
// Returns structured info about each discovered rule for codegen and list display.

import fs from 'node:fs'
import path from 'node:path'

export interface RuleMetadata {
  /** kebab-case rule name from // lintcn:name or derived from folder name */
  name: string
  /** runtime rule name parsed from Go `rule.Rule{Name: "..."}`. This is
   *  the name tsgolint uses in diagnostics and must match --warn flags.
   *  Falls back to `name` if the Go Name field can't be parsed. */
  goRuleName: string
  /** one-line description from // lintcn:description */
  description: string
  /** original source URL from // lintcn:source */
  source: string
  /** severity from // lintcn:severity — 'error' (default) or 'warn'.
   *  Warnings are displayed with yellow styling and don't cause exit code 1. */
  severity: 'error' | 'warn'
  /** exported Go variable name like NoFloatingPromisesRule */
  varName: string
  /** Go package name (= subfolder name, e.g. no_floating_promises) */
  packageName: string
}

// Matches `var XxxRule = rule.Rule{` with optional leading whitespace
// and optional import alias (e.g. `r.Rule{` if imported as `r "...rule"`)
const RULE_VAR_RE = /^\s*var\s+(\w+)\s*=\s*\w*\.?Rule\s*\{/m
const METADATA_RE = /^\/\/\s*lintcn:(\w+)\s+(.+)$/gm
// buildGoRuleNameRe creates a regex scoped to a specific rule variable's
// struct literal, e.g. `var FooRule = rule.Rule{ ... Name: "foo" ... }`.
// This avoids matching a Name field in an unrelated struct earlier in the file.
function buildGoRuleNameRe(varName: string): RegExp {
  return new RegExp(`var\\s+${varName}\\s*=\\s*\\w*\\.?Rule\\s*\\{[\\s\\S]*?Name:\\s*"([^"]+)"`)
}

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

/** Extract the Name field from a specific rule.Rule variable's struct literal.
 *  Scoped to varName to avoid matching Name fields in unrelated structs. */
export function parseGoRuleName(content: string, varName: string): string | undefined {
  const match = content.match(buildGoRuleNameRe(varName))
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

      const severity = meta.severity === 'warn' ? 'warn' as const : 'error' as const
      const displayName = meta.name || entry.name.replace(/_/g, '-')
      const goRuleName = parseGoRuleName(content, varName) || displayName

      rules.push({
        name: displayName,
        goRuleName,
        description: meta.description || '',
        source: meta.source || '',
        severity,
        varName,
        packageName: entry.name,
      })
    }
  }

  return rules
}
