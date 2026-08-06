/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyRedundantErrorWrap(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.SimplifyRedundantErrorWrap{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(err error) error {
				return fmt.Errorf("%w", err)
			}
		`, `
			package main

			func f(err error) error {
				return err
			}
		`),
	)
}

func TestSimplifyRedundantErrorWrapNoChangeWithContext(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.SimplifyRedundantErrorWrap{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(err error) error {
				return fmt.Errorf("failed to open: %w", err)
			}
		`),
	)
}

// Skips a non-error (any) argument, where replacing fmt.Errorf with the bare
// value would not compile.
func TestSimplifyRedundantErrorWrapNoChangeNonError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.SimplifyRedundantErrorWrap{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(x any) error {
				return fmt.Errorf("%w", x)
			}
		`),
	)
}
