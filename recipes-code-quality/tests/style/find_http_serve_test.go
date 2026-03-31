/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindHttpListenAndServe(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHttpListenAndServe{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				http.ListenAndServe(":8080", nil)
			}
		`),
	)
}

func TestFindHttpListenAndServeNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHttpListenAndServe{})
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
