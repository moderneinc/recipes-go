/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindLogFatal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindLogFatal{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "log"

			func main() {
				log.Fatal("error")
			}
		`),
	)
}

func TestFindLogFatalNoChangePrintln(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindLogFatal{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "log"

			func main() {
				log.Println("info")
			}
		`),
	)
}
