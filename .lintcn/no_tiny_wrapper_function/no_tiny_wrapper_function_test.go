// Tests for the no-tiny-wrapper-function rule.
package no_tiny_wrapper_function

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoTinyWrapperFunction(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(
		fixtures.GetRootDir(),
		"tsconfig.minimal.json",
		t,
		&NoTinyWrapperFunctionRule,
		validCases,
		invalidCases,
	)
}

var validCases = []rule_tester.ValidTestCase{
	{Code: `function sum(a: number, b: number) { return a + b }`},
	{Code: `const getName = (user: { name: string }) => user.name`},
	{Code: `class Counter { next() { return this.value + 1 } value = 0 }`},
	{Code: `function compute() { const value = 1; const next = value + 1; return next }`},
	{Code: `const build = () => ({ id: 1 })`},
	{Code: `declare function write(value: string): string; declare function normalize(value: string): string; function persist(value: string) { return write(normalize(value)) }`},
	{Code: `declare function write(value: string): string; function persist(user: { name: string }) { return write(user.name) }`},
	{Code: `declare function createMemoryRedis(): { url: string }; class ConversationStore { constructor(redis: { url: string }) {} } export function createTestStore(): ConversationStore { return new ConversationStore(createMemoryRedis()) }`},
	{Code: `
		declare const items: string[]
		declare function normalize(value: string): string

		const result = items.map((item) => normalize(item))
	`},
	{Code: `
		declare const items: string[]
		declare function isVisible(value: string): boolean

		const result = items.filter((item) => isVisible(item))
	`},
	{Code: `
		declare const items: string[]
		declare function store(value: string): void

		items.forEach((item) => store(item))
	`},
	{Code: `
		declare function flush(): void

		setTimeout(() => flush(), 10)
	`},
	{Code: `
		declare function run<T>(value: Promise<T>): void
		declare const errore: {
			tryAsync<T>(fn: () => Promise<T>): Promise<T>
		}
		declare const thread: {
			sendTyping(): Promise<void>
		}

		run(errore.tryAsync(() => thread.sendTyping()))
	`},
	{Code: `
		declare function debounce(options: {
			callback: () => Promise<void>
		}): void
		declare const service: {
			persist(): Promise<void>
		}

		debounce({
			callback: async () => service.persist(),
		})
	`},
	{Code: `
		declare function onClick(handler: () => void): void
		declare const service: {
			submit(): void
		}

		onClick(() => service.submit())
	`},
	{Code: `
		declare function prepare(): void;
		declare function write(value: string): string;
		declare function finalize(value: string): string;
		function wrapper(value: string) {
			prepare();
			const result = write(value);
			finalize(result);
			return result;
		}
	`},
	{Code: `class Box { format() { return this.value.toUpperCase } value = "x" }`},
}

var invalidCases = []rule_tester.InvalidTestCase{
	{
		Code: `
			class ConversationStore {
				constructor(redis: { url: string }) {}
			}
			export function createTestStore(redis: { url: string }): ConversationStore {
				return new ConversationStore(redis);
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "tinyWrapper"}},
	},
	{
		Code:   `declare function write(file: string): void; const save = (file: string) => write(file);`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "tinyWrapper"}},
	},
	{
		Code:   `class Repo { persist(value: string) { return write(value) } } declare function write(value: string): string;`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "tinyWrapper"}},
	},
	{
		Code: `
			declare function write(value: string): Promise<void>;
			class Service {
				async persist(value: string) {
					return await write(value);
				}
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "tinyWrapper"}},
	},
}
