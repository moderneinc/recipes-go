/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseCustomHttpClientGet(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseCustomHttpClient{})
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

func TestUseCustomHttpClientPost(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseCustomHttpClient{})
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

func TestUseCustomHttpClientNoChangeCustomClient(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseCustomHttpClient{})
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
