/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/errorhandling"
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

// Skips a void Close(), where `_ = t.Close()` would not compile.
func TestCheckCloseErrorNoChangeVoidClose(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.CheckCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type T struct{}

			func (T) Close() {}

			func f(t T) {
				t.Close()
			}
		`),
	)
}

// Skips a returned Close(), where `return _ = r.Close()` would not compile.
func TestCheckCloseErrorNoChangeReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.CheckCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(r *os.File) error {
				return r.Close()
			}
		`),
	)
}

// Skips a Close() whose error is already inspected in a condition.
func TestCheckCloseErrorNoChangeInCondition(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.CheckCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(r *os.File) {
				if r.Close() != nil {
					return
				}
			}
		`),
	)
}

// Skips a deferred Close(), where `defer _ = r.Close()` would not compile.
func TestCheckCloseErrorNoChangeDefer(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.CheckCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(r *os.File) {
				defer r.Close()
			}
		`),
	)
}
