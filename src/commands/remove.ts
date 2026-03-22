// lintcn remove <name> — delete a rule subfolder from .lintcn/

import fs from 'node:fs'
import path from 'node:path'
import { requireLintcnDir } from '../paths.ts'
import { discoverRules } from '../discover.ts'

export function removeRule(name: string): void {
  const lintcnDir = requireLintcnDir()
  const rules = discoverRules(lintcnDir)
  const normalizedName = name.replace(/-/g, '_')

  const match = rules.find((r) => {
    return r.name === name || r.packageName === normalizedName
  })

  if (!match) {
    throw new Error(
      `Rule "${name}" not found. Run \`lintcn list\` to see installed rules.`,
    )
  }

  // Remove the entire subfolder
  const ruleDir = path.join(lintcnDir, match.packageName)
  fs.rmSync(ruleDir, { recursive: true })
  console.log(`Removed ${match.packageName}/`)
}
