/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidOsExit(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AvoidOsExit{})
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

func TestAvoidOsExitZeroRemoved(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AvoidOsExit{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				println("done")
				os.Exit(0)
			}
		`, `
			package main

			import "os"

			func main() {
				println("done")
			}
		`),
	)
}

func TestAvoidOsExitNoChangeGetenv(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AvoidOsExit{})
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
