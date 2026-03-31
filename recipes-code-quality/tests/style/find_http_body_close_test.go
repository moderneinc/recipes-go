/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindHttpResponseBodyGet(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHttpResponseBody{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				http.Get("http://example.com")
			}
		`),
	)
}

func TestFindHttpResponseBodyNoChangeError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHttpResponseBody{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f(w http.ResponseWriter) {
				http.Error(w, "err", 500)
			}
		`),
	)
}
