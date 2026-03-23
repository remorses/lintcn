#!/usr/bin/env node

// lintcn — the shadcn for type-aware TypeScript lint rules.
// Add rules by URL, compile, and run them via tsgolint.

import { goke } from 'goke'
import { createRequire } from 'node:module'
import { addRule } from './commands/add.ts'
import { lint, buildBinary } from './commands/lint.ts'
import { listRules } from './commands/list.ts'
import { removeRule } from './commands/remove.ts'
import { clean } from './commands/clean.ts'
import { DEFAULT_TSGOLINT_VERSION } from './cache.ts'

const require = createRequire(import.meta.url)
const packageJson = require('../package.json') as { version: string }

const cli = goke('lintcn')

cli
  .command('add <url>', 'Add a rule by GitHub URL. Fetches the whole folder into .lintcn/{rule}/')
  .example('# Add a rule folder')
  .example('lintcn add https://github.com/oxc-project/tsgolint/tree/main/internal/rules/no_floating_promises')
  .example('# Add by file URL (auto-fetches the whole folder)')
  .example('lintcn add https://github.com/oxc-project/tsgolint/blob/main/internal/rules/await_thenable/await_thenable.go')
  .action(async (url) => {
    await addRule(url)
  })

cli
  .command('remove <name>', 'Remove an installed rule from .lintcn/')
  .example('lintcn remove no-floating-promises')
  .action((name) => {
    removeRule(name)
  })

cli
  .command('list', 'List all installed rules')
  .action(() => {
    listRules()
  })

cli
  .command('lint', 'Build custom tsgolint binary and run it against the project')
  .option('--rebuild', 'Force rebuild even if cached binary exists')
  .option('--fix', 'Automatically fix violations')
  .option('--tsconfig <path>', 'Path to tsconfig.json')
  .option('--list-files', 'List matched files')
  .option('--all-warnings', 'Show warnings for all files, not just git-changed ones')
  .option('--tsgolint-version [version]', 'Override the pinned tsgolint version (tag or commit). For testing unreleased tsgolint versions.')
  .action(async (options) => {
    const tsgolintVersion = (options.tsgolintVersion as string) || DEFAULT_TSGOLINT_VERSION
    const passthroughArgs: string[] = []
    if (options.fix) {
      passthroughArgs.push('--fix')
    }
    if (options.tsconfig) {
      passthroughArgs.push('--tsconfig', options.tsconfig as string)
    }
    if (options.listFiles) {
      passthroughArgs.push('--list-files')
    }
    // pass through anything after --
    const doubleDash = options['--']
    if (doubleDash && Array.isArray(doubleDash)) {
      passthroughArgs.push(...doubleDash)
    }
    const exitCode = await lint({
      rebuild: !!options.rebuild,
      tsgolintVersion,
      passthroughArgs,
      allWarnings: !!options.allWarnings,
    })
    process.exit(exitCode)
  })

cli
  .command('build', 'Build the custom tsgolint binary without running it')
  .option('--rebuild', 'Force rebuild even if cached binary exists')
  .option('--tsgolint-version [version]', 'Override the pinned tsgolint version (tag or commit). For testing unreleased tsgolint versions.')
  .action(async (options) => {
    const tsgolintVersion = (options.tsgolintVersion as string) || DEFAULT_TSGOLINT_VERSION
    const binaryPath = await buildBinary({ rebuild: !!options.rebuild, tsgolintVersion })
    console.log(binaryPath)
  })

cli
  .command('clean', 'Remove cached tsgolint source and compiled binaries to free disk space')
  .action(() => {
    clean()
  })

cli.help()
cli.version(packageJson.version)
cli.parse()
