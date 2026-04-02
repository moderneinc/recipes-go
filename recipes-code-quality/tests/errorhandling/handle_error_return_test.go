/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestHandleErrorReturnDiscarded(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleErrorReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				_, _ = os.Open("file")
			}
		`, `
			package main

			import "os"

			func main() {
				_, err = os.Open("file")
			}
		`),
	)
}

func TestHandleErrorReturnNotDiscarded(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleErrorReturn{})
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
