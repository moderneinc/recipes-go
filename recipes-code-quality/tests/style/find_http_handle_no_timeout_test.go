/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindHTTPListenAndServe(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHTTPListenAndServe{})
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

func TestFindHTTPListenAndServeNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHTTPListenAndServe{})
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
