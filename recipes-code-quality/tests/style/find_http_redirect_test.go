/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindHttpRedirect(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHttpRedirect{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func handler(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/login", 302)
			}
		`),
	)
}

func TestFindHttpRedirectNoChangeHttpError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHttpRedirect{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func handler(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "fail", 500)
			}
		`),
	)
}
