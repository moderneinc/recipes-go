/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestHandleDeferredCloseError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleDeferredCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func main() {
				f, _ := os.Open("file.txt")
				defer f.Close()
			}
		`, `
			package main

			import "os"

			func main() {
				f, _ := os.Open("file.txt")
				defer func() { _ = f.Close() }()
			}
		`),
	)
}

func TestHandleDeferredCloseErrorNoChangeDone(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleDeferredCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sync"

			func main() {
				var wg sync.WaitGroup
				wg.Add(1)
				defer wg.Done()
			}
		`),
	)
}

// Skips a void Close(), where `_ = t.Close()` inside the closure would not compile.
func TestHandleDeferredCloseErrorNoChangeVoidClose(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.HandleDeferredCloseError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type T struct{}

			func (T) Close() {}

			func f(t T) {
				defer t.Close()
			}
		`),
	)
}
