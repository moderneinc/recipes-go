/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindOsExit(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindOsExit{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				os.Exit(1)
			}
		`),
	)
}

func TestFindOsExitNoChangeGetenv(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindOsExit{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				_ = os.Getenv("PATH")
			}
		`),
	)
}
