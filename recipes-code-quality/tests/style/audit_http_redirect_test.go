/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditHttpRedirect(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditHttpRedirect{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func handler(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/login", 302)
			}
		`, `
			package main

			import "net/http"

			func handler(w http.ResponseWriter, r *http.Request) {/*~~(review redirect target and status code)~~>*/
				http.Redirect(w, r, "/login", 302)
			}
		`),
	)
}

func TestAuditHttpRedirectNoChangeHttpError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditHttpRedirect{})
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
