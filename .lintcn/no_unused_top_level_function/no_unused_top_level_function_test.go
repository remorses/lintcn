package no_unused_top_level_function

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoUnusedTopLevelFunction(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(
		fixtures.GetRootDir(),
		"tsconfig.minimal.json",
		t,
		&NoUnusedTopLevelFunctionRule,
		validCases,
		invalidCases,
	)
}

var validCases = []rule_tester.ValidTestCase{
	{Code: `function used() { return 1 } const value = used();`},
	{Code: `export function publicApi() { return 1 }`},
	{
		Code: `function shared() { return 1 } export { shared }`,
		Files: map[string]string{
			"index.ts": `import { shared } from "./file"; const value = shared();`,
		},
	},
	{
		Code: `const helper = () => 1;`,
	},
	{Code: `function outer() { function inner() { return 1 } return inner() } const value = outer()`},
	{
		Code: `function callback() { return 1 } export { callback }`,
		Files: map[string]string{
			"consumer.ts": `import { callback } from "./file"; const handlers = [callback]; handlers[0]();`,
		},
	},
}

var invalidCases = []rule_tester.InvalidTestCase{
	{
		Code: `function orphan() { return 1 }`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unusedTopLevelFunction"},
		},
	},
	{
		Code: `function helper() { return 1 }`,
		Files: map[string]string{
			"index.ts": `const value = 1;`,
		},
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unusedTopLevelFunction"},
		},
	},
	{
		Code: `function loop(value: number): number { if (value <= 0) return 0; return loop(value - 1) }`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "unusedTopLevelFunction"},
		},
	},
}
