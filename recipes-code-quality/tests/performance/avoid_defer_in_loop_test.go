/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindDeferInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AvoidDeferInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(files []string) {
				for _, name := range files {
					f, _ := os.Open(name)
					defer f.Close()
				}
			}
		`),
	)
}

func TestFindDeferInClassicForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AvoidDeferInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				for i := 0; i < 10; i++ {
					f, _ := os.Create("file")
					defer f.Close()
				}
			}
		`),
	)
}

func TestFindDeferNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AvoidDeferInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				f, _ := os.Open("file")
				defer f.Close()
			}
		`),
	)
}
