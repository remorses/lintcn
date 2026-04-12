// Tests for the no-redundant-exported-return-type rule.
package no_redundant_exported_return_type

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestNoRedundantExportedReturnType(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(
		fixtures.GetRootDir(),
		"tsconfig.minimal.json",
		t,
		&NoRedundantExportedReturnTypeRule,
		validCases,
		invalidCases,
	)
}

var validCases = []rule_tester.ValidTestCase{
	{
		Code: `
			type User = { id: string }
			export function getUser(): User {
				return { id: '1' }
			}
			type UserResult = ReturnType<typeof getUser>
		`,
	},
	{
		Code: `
			type User = { id: string }
			export function getUser(): User {
				return { id: '1' }
			}
			export type UserResult = ReturnType<typeof getUser>
		`,
	},
	{
		Code: `
			export function getUser() {
				return { id: '1' }
			}
			export type UserResult = ReturnType<typeof getUser>
		`,
	},
	{
		Code: `
			type PrivateUser = { id: string }
			export function getUser(): PrivateUser {
				return { id: '1' }
			}
			export type UserResult = ReturnType<typeof getUser>
		`,
	},
	{
		Code: `
			export function getUser(): { id: string } {
				return { id: '1' }
			}
			export type UserResult = ReturnType<typeof getUser>
		`,
	},
	{
		Code: `
			export interface User { id: string }
			type Loader = () => User
			export type UserResult = ReturnType<Loader>
		`,
	},
	{
		Code: `
			type PrivateUser = { id: string }
			export type Loader = () => PrivateUser
			export type UserResult = ReturnType<Loader>
		`,
	},
	{
		Code: `
			export interface User { id: string }
			export function getUser(): Promise<PrivateUser> {
				return Promise.resolve({ id: '1' })
			}
			type PrivateUser = User
			export type UserResult = ReturnType<typeof getUser>
		`,
	},
	{
		Code: `
			export interface User { id: string }
			function getUser(): User {
				return { id: '1' }
			}
			declare function takesUser(user: ReturnType<typeof getUser>): void
		`,
	},
	{
		Code: `
			export interface User { id: string }
			export function getUser(): PrivateUser {
				return { id: '1' }
			}
			type PrivateUser = User
			declare function takesUser(user: ReturnType<typeof getUser> | null): void
		`,
	},
	{
		Code: `
			export interface User { id: string }
			export type Loader = () => PrivateUser
			type PrivateUser = User
			interface Config {
				load: ReturnType<Loader>
			}
		`,
	},
	{
		Code: `
			import { getUser } from './api'
			type Wrapped = {
				value: ReturnType<typeof getUser>
			}
		`,
		Files: map[string]string{
			"api.ts": `
				type PrivateUser = { id: string }
				export function getUser(): PrivateUser {
					return { id: '1' }
				}
			`,
		},
	},
	{
		Code: `
			import { getSession } from 'auth-kit'
			declare function takeSession(session: ReturnType<typeof getSession>): void
		`,
		Files: map[string]string{
			"node_modules/auth-kit/package.json": `{"name":"auth-kit","types":"index.d.ts"}`,
			"node_modules/auth-kit/index.d.ts": `
				export interface Session {
					userId: string
				}
				export function getSession(): { userId: string }
			`,
		},
	},
	{
		Code: `
			import { getUser } from './api'
			type Wrapped = Promise<ReturnType<typeof getUser>>
		`,
		Files: map[string]string{
			"api.ts": `
				export function getUser(): { id: string } {
					return { id: '1' }
				}
			`,
		},
	},
}

var invalidCases = []rule_tester.InvalidTestCase{
	{
		Code: `
			export interface User { id: string }
			export function getUser(): User {
				return { id: '1' }
			}
			export type UserResult = ReturnType<typeof getUser>
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			interface User { id: string }
			export { User }
			export function getUser(): User {
				return { id: '1' }
			}
			export type UserResult = ReturnType<typeof getUser>
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			export interface Box<T> { value: T }
			export type Loader<T> = () => Box<T>
			export type StringBox = ReturnType<Loader<string>>
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			export interface User { id: string }
			export type Loader = () => User
			export type UserResult = ReturnType<Loader>
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			export interface User { id: string }
			export function getUser(): Promise<User> {
				return Promise.resolve({ id: '1' })
			}
			export type UserResult = ReturnType<typeof getUser>
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			export interface User { id: string }
			export function getUser(): User | null {
				return { id: '1' }
			}
			export type UserResult = ReturnType<typeof getUser>
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			export interface User { id: string }
			export type Loader<T> = () => Promise<T>
			export type UserPromise = ReturnType<Loader<User>>
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			export interface User { id: string }
			export interface Box<T> { value: T }
			export type Loader<T> = () => Promise<Box<T>>
			export type UserPromise = ReturnType<Loader<User>>
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			export interface User { id: string }
			export function getUser(): User {
				return { id: '1' }
			}
			declare function takesUser(user: ReturnType<typeof getUser>): void
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			export interface User { id: string }
			export function getUser(): User {
				return { id: '1' }
			}
			export function wrap(user: User): User {
				return user
			}
			function saveUser(user: ReturnType<typeof getUser>): ReturnType<typeof wrap> {
				return wrap(user)
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{
			{MessageId: "redundantExportedReturnType"},
			{MessageId: "redundantExportedReturnType"},
		},
	},
	{
		Code: `
			export interface User { id: string }
			export function getUser(): User {
				return { id: '1' }
			}
			type MaybeUser = ReturnType<typeof getUser> | null
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			export interface User { id: string }
			export interface Tagged { tag: string }
			export function getUser(): User {
				return { id: '1' }
			}
			type TaggedUser = ReturnType<typeof getUser> & Tagged
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			export interface User { id: string }
			export function getUser(): User {
				return { id: '1' }
			}
			interface Config {
				currentUser: ReturnType<typeof getUser>
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			export interface User { id: string }
			export type Loader = () => User
			interface Config {
				currentUser: ReturnType<Loader>
			}
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			import type { User } from './types'
			export function getUser(): User {
				return { id: '1' }
			}
			declare function takesUser(user: ReturnType<typeof getUser>): void
		`,
		Files: map[string]string{
			"types.ts": `export interface User { id: string }`,
		},
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			import { getUser } from './api'
			type Nested = {
				meta: {
					user: ReturnType<typeof getUser>
				}
			}
		`,
		Files: map[string]string{
			"api.ts": `
				export interface User {
					id: string
				}
				export function getUser(): User {
					return { id: '1' }
				}
			`,
		},
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			import { getUser } from './api'
			type UserTuple = [ReturnType<typeof getUser>, string]
		`,
		Files: map[string]string{
			"api.ts": `
				export interface User {
					id: string
				}
				export function getUser(): User {
					return { id: '1' }
				}
			`,
		},
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			import { getUser } from './api'
			type UserList = Array<ReturnType<typeof getUser>>
		`,
		Files: map[string]string{
			"api.ts": `
				export interface User {
					id: string
				}
				export function getUser(): User {
					return { id: '1' }
				}
			`,
		},
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			import { getUser } from './api'
			declare function takeUser(user: ReturnType<typeof getUser>): void
		`,
		Files: map[string]string{
			"api.ts": `
				export { User } from './types'
				export function getUser(): User {
					return { id: '1' }
				}
				import type { User } from './types'
			`,
			"types.ts": `
				export interface User {
					id: string
				}
			`,
		},
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
	{
		Code: `
			import { getSession } from 'auth-kit'
			declare function takeSession(session: ReturnType<typeof getSession>): void
		`,
		Files: map[string]string{
			"node_modules/auth-kit/package.json": `{"name":"auth-kit","types":"index.d.ts"}`,
			"node_modules/auth-kit/index.d.ts": `
				export interface User {
					id: string
				}
				export function getSession(): User
			`,
		},
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "redundantExportedReturnType"}},
	},
}
