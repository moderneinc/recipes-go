/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseTlsForHttp(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseTlsForHttp{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				http.ListenAndServe(":8080", nil)
			}
		`, `
			package main

			import "net/http"

			func f() {
				http.ListenAndServeTLS(":8080", "cert.pem", "key.pem", nil)
			}
		`),
	)
}

func TestUseTlsForHttpNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseTlsForHttp{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				http.ListenAndServeTLS(":443", "cert.pem", "key.pem", nil)
			}
		`),
	)
}
