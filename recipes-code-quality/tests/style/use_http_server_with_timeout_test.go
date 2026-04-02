/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseHttpServerWithTimeout(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseHttpServerWithTimeout{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func main() {
				http.ListenAndServe(":8080", nil)
			}
		`),
	)
}

func TestUseHttpServerWithTimeoutNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseHttpServerWithTimeout{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func main() {
				server := &http.Server{Addr: ":8080"}
				server.ListenAndServe()
			}
		`),
	)
}
