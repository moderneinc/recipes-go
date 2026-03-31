/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindPanicSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindPanic{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {
				panic("oops")
			}
		`),
	)
}

func TestFindPanicNoChangeNoPanic(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindPanic{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func main() {
				fmt.Println("hello")
			}
		`),
	)
}

func TestFindPanicWithVariable(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindPanic{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {
				err := recover()
				if err != nil {
					panic(err)
				}
			}
		`),
	)
}
