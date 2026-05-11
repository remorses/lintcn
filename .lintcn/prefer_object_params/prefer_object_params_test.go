// Tests for the prefer-object-params rule.
package prefer_object_params

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestPreferObjectParams(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(
		fixtures.GetRootDir(),
		"tsconfig.minimal.json",
		t,
		&PreferObjectParamsRule,
		validCases,
		invalidCases,
	)
}

var validCases = []rule_tester.ValidTestCase{
	{
		Code: `
			function createUser(name: string, email: string) {
				return { name, email }
			}
		`,
	},
	{
		Code: `
			function createUser(options: {
				name: string
				email: string
				role: string
			}) {
				return options
			}
		`,
	},
	{
		Code: `
			declare const items: string[]

			items.map((value, index, array) => {
				return value + index + array.length
			})
		`,
	},
	{
		Code: `
			declare function register(config: {
				format: (value: string, index: number, array: string[]) => string
			}): void

			register({
				format: (value, index, array) => {
					return value + index + array.length
				},
			})
		`,
	},
	{
		Code: `
			declare function register(config: {
				format(value: string, index: number, array: string[]): string
			}): void

			register({
				format(value, index, array) {
					return value + index + array.length
				},
			})
		`,
	},
	{
		Code: `
			const value = ((left: number, right: number, extra: number) => {
				return left + right + extra
			})(1, 2, 3)
		`,
	},
	{
		Code: `
			class UserService {
				update(name: string, email: string) {
					return { name, email }
				}
			}
		`,
	},
	{
		Code: `
			declare function createUser(name: string, email: string, role: string): void
		`,
	},
	{
		Code: `
			const handlers = {
				update(name: string, email: string) {
					return { name, email }
				},
			}
		`,
	},
	{
		Code: `
			const createUser = (name: string, email: string) => {
				return { name, email }
			}
		`,
	},
	{
		Code: `
			function updateUser(this: { id: string }, name: string, email: string) {
				return { id: this.id, name, email }
			}
		`,
	},
}

var invalidCases = []rule_tester.InvalidTestCase{
	{
		Code: `
			function createUser(name: string, email: string, role: string) {
				return { name, email, role }
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferObjectParams"}},
	},
	{
		Code: `
			function createUser(name: string, email: string, role: string, active: boolean) {
				return { name, email, role, active }
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferObjectParams"}},
	},
	{
		Code: `
			const createUser = (name: string, email: string, role: string) => {
				return { name, email, role }
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferObjectParams"}},
	},
	{
		Code: `
			const createUser = function (name: string, email: string, role: string) {
				return { name, email, role }
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferObjectParams"}},
	},
	{
		Code: `
			class UserService {
				update(name: string, email: string, role: string) {
					return { name, email, role }
				}
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferObjectParams"}},
	},
	{
		Code: `
			class UserService {
				constructor(name: string, email: string, role: string) {}
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferObjectParams"}},
	},
	{
		Code: `
			const handlers = {
				update(name: string, email: string, role: string) {
					return { name, email, role }
				},
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferObjectParams"}},
	},
	{
		Code: `
			const handlers = {
				update: (name: string, email: string, role: string) => {
					return { name, email, role }
				},
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferObjectParams"}},
	},
	{
		Code: `
			declare const captureRejectionSymbol: unique symbol

			class Http2Session {
				[captureRejectionSymbol](err: Error, event: string, args: unknown[]) {
					return { err, event, args }
				}
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferObjectParams"}},
	},
	{
		Code: `
			declare const key: unique symbol

			const handlers = {
				[key]: (name: string, email: string, role: string) => {
					return { name, email, role }
				},
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferObjectParams"}},
	},
}
