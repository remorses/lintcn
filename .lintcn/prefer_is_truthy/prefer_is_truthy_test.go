// Tests for the prefer-is-truthy rule.
package prefer_is_truthy

import (
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
	"github.com/typescript-eslint/tsgolint/internal/rules/fixtures"
)

func TestPreferIsTruthy(t *testing.T) {
	t.Parallel()
	rule_tester.RunRuleTester(
		fixtures.GetRootDir(),
		"tsconfig.minimal.json",
		t,
		&PreferIsTruthyRule,
		validCases,
		invalidCases,
	)
}

var validCases = []rule_tester.ValidTestCase{
	{
		Code: `
			declare const pokemon: (Pokemon | null | undefined)[]
			declare function isTruthy<T>(value: T | null | undefined): value is T
			interface Pokemon {
				name: string
			}

			const activePokemon = pokemon.filter(isTruthy)
		`,
	},
	{
		Code: `
			declare const pokemon: (Pokemon | null | undefined)[]
			interface Pokemon {
				name: string
			}

			const activePokemon = pokemon.filter((value) => value !== null)
		`,
	},
	{
		Code: `
			declare const pokemon: (Pokemon | null | undefined)[]
			interface Pokemon {
				name: string
			}

			const isPokemon = (value: Pokemon | null | undefined): value is Pokemon => value != null
			const activePokemon = pokemon.filter(isPokemon)
		`,
	},
	{
		Code: `
			declare const pokemon: Pokemon[]
			interface Pokemon {
				name: string
			}

			const activePokemon = pokemon.filter((value): value is Pokemon => value.name.length > 0)
		`,
	},
	{
		Code: `
			declare const pokemon: (Pokemon | null)[]
			interface Pokemon {
				name: string
			}

			const activePokemon = pokemon.map((value): value is Pokemon => value !== null)
		`,
	},
}

var invalidCases = []rule_tester.InvalidTestCase{
	{
		Code: `
			declare const pokemon: (Pokemon | null | undefined)[]
			interface Pokemon {
				name: string
			}

			const activePokemon = pokemon.filter((value): value is Pokemon => value !== null)
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferIsTruthy"}},
	},
	{
		Code: `
			declare const pokemon: (Pokemon | null | undefined)[]
			interface Pokemon {
				name: string
			}

			const activePokemon = pokemon.filter(function (value): value is Pokemon {
				return value != null
			})
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferIsTruthy"}},
	},
	{
		Code: `
			declare const pokemon: (Pokemon | null | undefined)[]
			interface Pokemon {
				name: string
			}

			const activePokemon = pokemon.filter((value): value is Pokemon => value !== null && value !== undefined)
		`,
		Errors: []rule_tester.InvalidTestCaseError{{MessageId: "preferIsTruthy"}},
	},
}
