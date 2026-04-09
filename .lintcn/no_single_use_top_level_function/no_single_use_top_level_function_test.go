package no_single_use_top_level_function

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoSingleUseTopLevelFunction(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(
		fixtures.GetRootDir(),
		"tsconfig.minimal.json",
		t,
		&NoSingleUseTopLevelFunctionRule,
		validCases,
		invalidCases,
	)
}

var validCases = []rule_tester.ValidTestCase{
	{Code: `function helper() { return 1 }`},
	{Code: `function helper() { return 1 } const a = helper(); const b = helper();`},
	{Code: `export function publicApi() { return 1 } const value = publicApi();`},
	{Code: `function outer() { function inner() { return 1 } return inner() } const a = outer(); const b = outer();`},
	{Code: `const helper = () => 1; const value = helper();`},
	{Code: `const helper = (value: number) => value + 1; const result = helper(1)`},
	{Code: `export const helper = () => 1; const value = helper()`},
	{Code: `declare function subscribe(fn: () => void): void; subscribe(() => { console.log('hi') })`},
	{Code: `const wrap = () => { return () => { console.log('hi') } }; const fn = wrap(); fn()`},
	{Code: `function createLogger() { return () => { console.log('hi') } } const a = createLogger(); const b = createLogger(); a(); b()`},
	{Code: `const helper = function () { return 1 }; const value = helper();`},
	{Code: `const helper = function namedHelper() { return 1 }; const value = helper();`},
	{Code: `let helper = function () { return 1 }; const value = helper();`},
	{Code: `const value = (() => 1)()`},
	{Code: `const value = (function () { return 1 })()`},
	{Code: `const tools = { helper() { return 1 } }; const value = tools.helper()`},
	{Code: `const tools = { helper: () => 1 }; const value = tools.helper()`},
	{Code: `class Service { helper() { return 1 } } const service = new Service(); const value = service.helper()`},
	{Code: `function helper() { return 1 } export { helper } const value = helper();`},
	{Code: `function helper() { return 1 } const alias = helper; alias();`},
	{
		Code: `
			const TOAST_SESSION_ID_REGEX = /\[session:([^\]]+)\]$/

			function extractToastSessionId(message: string): string | undefined {
				const match = message.match(TOAST_SESSION_ID_REGEX)
				if (!match) {
					return undefined
				}
				return match[1]
			}

			const sessionId = extractToastSessionId('hello [session:abc]')
		`,
	},
	{
		Code: `
			const TOAST_SESSION_ID_REGEX = /\[session:[^\]]+\]$/

			function stripToastSessionId(message: string): string {
				return message.replace(TOAST_SESSION_ID_REGEX, '').trimEnd()
			}

			const cleanMessage = stripToastSessionId('hello [session:abc]')
		`,
	},
	{
		Code: `
			function verboseHelper(input: number) {
				const a = input + 1
				const b = a + 1
				const c = b + 1
				const d = c + 1
				const e = d + 1
				const f = e + 1
				const g = f + 1
				const h = g + 1
				const i = h + 1
				return i
			}

			const value = verboseHelper(1)
		`,
	},
	{
		Code: `function helper() { return 1 } export { helper }`,
		Files: map[string]string{
			"consumer.ts": `import { helper } from "./file"; const fn = helper; fn();`,
		},
	},
	{
		Code: `export function helper() { return 1 }`,
		Files: map[string]string{
			"consumer.ts": `import { helper as renamed } from "./file"; const value = renamed();`,
		},
	},
}

var invalidCases = []rule_tester.InvalidTestCase{
	{
		Code: `function helper() { return 1 } const value = helper()`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "singleUseTopLevelFunction"},
		},
	},
	{
		Code: `
			import path from 'node:path'

			type UserConfig = {
				build?: { outDir?: string }
				environments?: {
					client?: { build?: { outDir?: string } }
					rsc?: { build?: { outDir?: string } }
					ssr?: { build?: { outDir?: string } }
				}
				plugins?: string[]
			}

			declare function hasPluginNamed(plugins: string[] | undefined, name: string): boolean

			function getRootBuildOutDir(userConfig: UserConfig): string {
				return userConfig.build?.outDir ?? 'dist'
			}

			function getDefaultEnvironmentOutDirs(rootOutDir: string, isCloudflare: boolean) {
				return {
					client: path.join(rootOutDir, 'client'),
					rsc: path.join(rootOutDir, 'rsc'),
					ssr: isCloudflare ? path.join(rootOutDir, 'rsc/ssr') : path.join(rootOutDir, 'ssr'),
				}
			}

			function normalizeEnvironmentOutDirs(userConfig: UserConfig): UserConfig {
				const isCloudflare = hasPluginNamed(userConfig.plugins, 'vite-plugin-cloudflare')
				const rootOutDir = getRootBuildOutDir(userConfig)
				const defaults = getDefaultEnvironmentOutDirs(rootOutDir, isCloudflare)

				return {
					build: {
						outDir: rootOutDir,
					},
					environments: {
						client: {
							build: {
								outDir: userConfig.environments?.client?.build?.outDir ?? defaults.client,
							},
						},
						rsc: {
							build: {
								outDir: userConfig.environments?.rsc?.build?.outDir ?? defaults.rsc,
							},
						},
						ssr: {
							build: {
								outDir: userConfig.environments?.ssr?.build?.outDir ?? defaults.ssr,
							},
						},
					},
				}
			}

			const value = normalizeEnvironmentOutDirs({})
		`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "singleUseTopLevelFunction"},
			{MessageId: "singleUseTopLevelFunction"},
		},
	},
}
