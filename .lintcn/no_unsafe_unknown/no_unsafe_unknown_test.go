package no_unsafe_unknown

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoUnsafeUnknown(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(
		fixtures.GetRootDir(),
		"tsconfig.minimal.json",
		t,
		&NoUnsafeUnknownRule,
		validCases,
		invalidCases,
	)
}

var validCases = []rule_tester.ValidTestCase{
	{Code: `function greet(name: string) { return name }`},
	{Code: `const fn = (value: number) => value + 1`},
	{Code: `declare const value: unknown; const copy = value;`},
	{Code: `function parse(): unknown { return undefined }`},
	{Code: `const value = input as string;`},
	{Code: `const value = { a: 1 } as const;`},
	{Code: `function wrap<T>(value: T) { return value }`},
	{Code: `
		function parseErrorMessage(err: unknown): string | undefined {
			if (
				err &&
				typeof err === 'object' &&
				'message' in err &&
				typeof err.message === 'string'
			) {
				return err.message
			}
			return undefined
		}
	`},
	{Code: `
		function normalizeLogo(raw: unknown): string | undefined {
			if (!raw || typeof raw !== 'object') {
				return undefined
			}
			if ('src' in raw && typeof raw.src === 'string') {
				return raw.src
			}
			return undefined
		}
	`},
}

var invalidCases = []rule_tester.InvalidTestCase{
	{
		Code: `function run(input: unknown) { return input }`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unknownFunctionParameter"},
		},
	},
	{
		Code: `const run = (input: unknown[]) => input.length`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unknownFunctionParameter"},
		},
	},
	{
		Code: `function run(input: Promise<unknown>) { return input }`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unknownFunctionParameter"},
		},
	},
	{
		Code: `class Service { run(input: unknown) { return input } }`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unknownFunctionParameter"},
		},
	},
	{
		Code: `class Service { constructor(private readonly input: unknown) {} }`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unknownFunctionParameter"},
		},
	},
	{
		Code: `declare function map<T, U>(value: T, fn: (input: unknown) => U): U;`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unknownFunctionParameter"},
		},
	},
	{
		Code: `const value = input as unknown;`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unknownAssertion"},
		},
	},
	{
		Code: `const value = input as Promise<unknown>;`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unknownAssertion"},
		},
	},
	{
		Code: `const value = <unknown>input;`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unknownAssertion"},
		},
	},
	{
		Code: `declare const value: unknown; const result = value as string;`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unknownAssertionFromUnknown"},
		},
	},
	{
		Code: `declare const value: { id: string }; const result = (value as unknown) as Map<string, number>;`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unknownAssertionFromUnknown"},
			{MessageId: "unknownAssertion"},
		},
	},
}
