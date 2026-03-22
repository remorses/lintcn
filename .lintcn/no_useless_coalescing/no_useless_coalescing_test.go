package no_useless_coalescing

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoUselessCoalescing(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(fixtures.GetRootDir(), "tsconfig.json", t, &NoUselessCoalescingRule,
		[]rule_tester.ValidTestCase{
			// --- ?? valid cases ---

			// Nullish type with non-undefined fallback — useful
			{Code: `declare const x: string | undefined; x ?? 'default';`},
			// Null type with fallback — useful
			{Code: `declare const x: string | null; x ?? 'default';`},
			// Null | undefined with fallback
			{Code: `declare const x: string | null | undefined; x ?? 'default';`},
			// ?? undefined where type includes null — can't simplify
			{Code: `declare const x: string | null; x ?? undefined;`},
			// any type — can't reason statically
			{Code: `declare const x: any; x ?? 'default';`},
			// unknown type — can't reason statically
			{Code: `declare const x: unknown; x ?? 'default';`},
			// Type parameter — can't reason statically
			{Code: `function f<T>(x: T) { x ?? 'default'; }`},

			// --- || valid cases ---

			// String | undefined with string fallback — meaningful
			{Code: `declare const x: string | undefined; x || 'default';`},
			// Number | null with number fallback — meaningful
			{Code: `declare const x: number | null; x || 0;`},
			// String that could be empty, non-empty fallback — meaningful
			{Code: `declare const x: string; x || 'fallback';`},
			// Boolean with true fallback — meaningful coercion
			{Code: `declare const x: boolean; x || true;`},
			// any type — can't reason
			{Code: `declare const x: any; x || 'default';`},
			// unknown type — can't reason
			{Code: `declare const x: unknown; x || 'default';`},
			// string | null with || undefined — null is not undefined, so this normalizes
			{Code: `declare const x: string | null; x || undefined;`},
			// number | undefined — has non-nullish falsy potential (0)
			{Code: `declare const x: number | undefined; x || undefined;`},
		},
		[]rule_tester.InvalidTestCase{
			// --- ?? useless: left can never be nullish ---

			// String can never be nullish
			{
				Code:   `declare const x: string; x ?? 'fallback';`,
				Output: []string{`declare const x: string; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "uselessCoalescing"},
				},
			},
			// Number can never be nullish
			{
				Code:   `declare const x: number; x ?? 0;`,
				Output: []string{`declare const x: number; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "uselessCoalescing"},
				},
			},
			// Boolean can never be nullish
			{
				Code:   `declare const x: boolean; x ?? false;`,
				Output: []string{`declare const x: boolean; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "uselessCoalescing"},
				},
			},
			// Object can never be nullish
			{
				Code:   `declare const obj: { name: string }; obj ?? { name: 'fallback' };`,
				Output: []string{`declare const obj: { name: string }; obj;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "uselessCoalescing"},
				},
			},
			// Array can never be nullish
			{
				Code:   `declare const arr: string[]; arr ?? [];`,
				Output: []string{`declare const arr: string[]; arr;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "uselessCoalescing"},
				},
			},

			// --- ?? redundantUndefinedFallback ---

			// T | undefined with ?? undefined
			{
				Code:   `declare const x: string | undefined; x ?? undefined;`,
				Output: []string{`declare const x: string | undefined; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "redundantUndefinedFallback"},
				},
			},
			// T | undefined with ?? void 0
			{
				Code:   `declare const x: string | undefined; x ?? void 0;`,
				Output: []string{`declare const x: string | undefined; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "redundantUndefinedFallback"},
				},
			},
			// boolean | undefined with ?? undefined
			{
				Code:   `declare const x: boolean | undefined; x ?? undefined;`,
				Output: []string{`declare const x: boolean | undefined; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "redundantUndefinedFallback"},
				},
			},

			// --- || useless: left is always truthy ---

			// Object is always truthy
			{
				Code:   `declare const obj: { name: string }; obj || undefined;`,
				Output: []string{`declare const obj: { name: string }; obj;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "uselessCoalescing"},
				},
			},
			// Object with different fallback
			{
				Code:   `declare const obj: { name: string }; obj || { name: 'fallback' };`,
				Output: []string{`declare const obj: { name: string }; obj;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "uselessCoalescing"},
				},
			},

			// --- || identity fallback ---

			// string || '' — empty string identity
			{
				Code:   `declare const x: string; x || '';`,
				Output: []string{`declare const x: string; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "uselessCoalescing"},
				},
			},
			// boolean || false — false identity
			{
				Code:   `declare const x: boolean; x || false;`,
				Output: []string{`declare const x: boolean; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "uselessCoalescing"},
				},
			},
			// bigint || 0n — zero bigint identity
			{
				Code:   `declare const x: bigint; x || 0n;`,
				Output: []string{`declare const x: bigint; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "uselessCoalescing"},
				},
			},

			// --- || redundantUndefinedFallback ---

			// string[] | undefined — arrays are always truthy, undefined → undefined
			{
				Code:   `declare const x: string[] | undefined; x || undefined;`,
				Output: []string{`declare const x: string[] | undefined; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "redundantUndefinedFallback"},
				},
			},
			// object | undefined — object always truthy
			{
				Code:   `declare const x: { a: number } | undefined; x || undefined;`,
				Output: []string{`declare const x: { a: number } | undefined; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "redundantUndefinedFallback"},
				},
			},
			// Literal string union | undefined — 'hello' always truthy
			{
				Code:   `declare const x: 'hello' | undefined; x || undefined;`,
				Output: []string{`declare const x: 'hello' | undefined; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "redundantUndefinedFallback"},
				},
			},

			// --- edge cases ---

			// Const string — always truthy for ?? (never nullish)
			{
				Code:   `const myVar = 'hello'; myVar ?? undefined;`,
				Output: []string{`const myVar = 'hello'; myVar;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "uselessCoalescing"},
				},
			},
			// String literal type — never nullish
			{
				Code:   `declare const x: 'hello'; x ?? undefined;`,
				Output: []string{`declare const x: 'hello'; x;`},
				Errors: []rule_tester.InvalidTestCaseError{
					{MessageId: "uselessCoalescing"},
				},
			},
		})
}
