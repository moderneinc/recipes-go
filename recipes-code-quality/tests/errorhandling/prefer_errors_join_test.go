/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/errorhandling"
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

			import "fmt"

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
