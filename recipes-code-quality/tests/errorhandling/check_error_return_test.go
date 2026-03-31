/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestCheckErrorReturnDiscarded(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.CheckErrorReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				_, _ = os.Open("file")
			}
		`),
	)
}

func TestCheckErrorReturnNotDiscarded(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.CheckErrorReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				f, err := os.Open("file")
				_ = f
				_ = err
			}
		`),
	)
}
