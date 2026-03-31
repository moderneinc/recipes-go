/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindHttpDefaultClientGet(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHttpDefaultClient{})
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

func TestFindHttpDefaultClientPost(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHttpDefaultClient{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				http.Post("http://example.com", "text/plain", nil)
			}
		`),
	)
}

func TestFindHttpDefaultClientNoChangeCustomClient(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHttpDefaultClient{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				client := &http.Client{}
				client.Get("http://example.com")
			}
		`),
	)
}
