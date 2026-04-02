/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestCheckCloseErrorSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.CheckCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				f, _ := os.Open("file.txt")
				f.Close()
			}
		`, `
			package main

			import "os"

			func main() {
				f, _ := os.Open("file.txt")
				_ = f.Close()
			}
		`),
	)
}

func TestCheckCloseErrorRespBody(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.CheckCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(r *os.File) {
				r.Close()
			}
		`, `
			package main

			import "os"

			func f(r *os.File) {
				_ = r.Close()
			}
		`),
	)
}

func TestCheckCloseErrorNoChangeRead(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.CheckCloseError{})
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
