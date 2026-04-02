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
			declare function createMemoryRedis(): { url: string };
			class ConversationStore {
				constructor(redis: { url: string }) {}
			}
			export function createTestStore(): ConversationStore {
				return new ConversationStore(createMemoryRedis());
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
			declare function normalize(value: string): string;
			declare function write(value: string): void;
			function persist(value: string) {
				const normalized = normalize(value);
				write(normalized);
				return normalized;
			}
		`,
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
