/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindOsOpenInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindFileOpenInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				for i := 0; i < 10; i++ {
					_, _ = os.Open("file.txt")
				}
			}
		`),
	)
}

func TestFindOsOpenNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindFileOpenInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				_, _ = os.Open("file.txt")
			}
		`),
	)
}
