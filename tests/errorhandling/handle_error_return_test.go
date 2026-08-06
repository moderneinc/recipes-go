/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestHandleErrorReturnDiscarded(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleErrorReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() error {
				file, _ := os.Open("file")
				_ = file
				return nil
			}
		`, `
			package main

			import "os"

			func f() error {
				file, err := os.Open("file")
				if err != nil {
					return err
				}
				_ = file
				return nil
			}
		`),
	)
}

// Skips a plain `=` assignment in main(), where `err` is undeclared and no error return exists.
func TestHandleErrorReturnNoChangeUndeclaredErr(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleErrorReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				_, _ = os.Open("file")
			}
		`),
	)
}

// Skips the comma-ok map access, where the discarded value is a bool rather than an error.
func TestHandleErrorReturnNoChangeCommaOkMap(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleErrorReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(m map[string]int) error {
				v, _ := m["k"]
				_ = v
				return nil
			}
		`),
	)
}

// Skips the comma-ok type assertion, where the discarded value is a bool.
func TestHandleErrorReturnNoChangeCommaOkTypeAssert(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleErrorReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(i interface{}) error {
				s, _ := i.(string)
				_ = s
				return nil
			}
		`),
	)
}

// Skips a capture in a loop body, where the inserted `return err` would change control flow.
func TestHandleErrorReturnNoChangeInsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleErrorReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(names []string) error {
				for _, n := range names {
					file, _ := os.Open(n)
					_ = file
				}
				return nil
			}
		`),
	)
}

func TestHandleErrorReturnNotDiscarded(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleErrorReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				f, err := os.Open("file")
				_ = f
				_ = err
			}
		`),
	)
}
