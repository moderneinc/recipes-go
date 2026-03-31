/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindMultipleErrorWrapsFound(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindMultipleErrorWraps{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(err1, err2 error) error {
				return fmt.Errorf("a: %w, b: %w", err1, err2)
			}
		`),
	)
}

func TestFindMultipleErrorWrapsNoChangeSingleW(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindMultipleErrorWraps{})
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

func TestFindMultipleErrorWrapsNoChangeNoW(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindMultipleErrorWraps{})
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
