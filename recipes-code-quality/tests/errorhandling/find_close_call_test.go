/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindCloseCallSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindCloseCall{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				f, _ := os.Open("file.txt")
				f.Close()
			}
		`),
	)
}

func TestFindCloseCallRespBody(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindCloseCall{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(r *os.File) {
				r.Close()
			}
		`),
	)
}

func TestFindCloseCallNoChangeRead(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindCloseCall{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				f, _ := os.Open("file.txt")
				buf := make([]byte, 100)
				f.Read(buf)
			}
		`),
	)
}
