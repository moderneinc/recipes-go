/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidLogFatal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AvoidLogFatal{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "log"

			func main() {
				log.Fatal("error")
			}
		`, `
			package main

			import "log"

			func main() {
				log.Println("error")
			}
		`),
	)
}

func TestAvoidLogFatalNoChangePrintln(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AvoidLogFatal{})
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
