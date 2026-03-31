/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferErrorfWrapVerb(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorfWrapVerb{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(err error) error {
				return fmt.Errorf("failed: %s", err)
			}
		`, `
			package main

			import "fmt"

			func f(err error) error {
				return fmt.Errorf("failed: %w", err)
			}
		`),
	)
}

func TestPreferErrorfWrapVerbNoChangeAlreadyW(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorfWrapVerb{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(err error) error {
				return fmt.Errorf("failed: %w", err)
			}
		`),
	)
}

func TestPreferErrorfWrapVerbNoChangeNonErr(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorfWrapVerb{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(msg string) error {
				return fmt.Errorf("failed: %s", msg)
			}
		`),
	)
}
