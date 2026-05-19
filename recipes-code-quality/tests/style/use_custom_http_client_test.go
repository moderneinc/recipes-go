/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
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
		`, `
			package main

			import "net/http"

			func f() {
				http.DefaultClient.Get("http://example.com")
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
		`, `
			package main

			import "net/http"

			func f() {
				http.DefaultClient.Post("http://example.com", "text/plain", nil)
			}
		`),
	)
}

func TestUseCustomHttpClientHead(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseCustomHttpClient{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				http.Head("http://example.com")
			}
		`, `
			package main

			import "net/http"

			func f() {
				http.DefaultClient.Head("http://example.com")
			}
		`),
	)
}

func TestUseCustomHttpClientPostForm(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseCustomHttpClient{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"net/http"
				"net/url"
			)

			func f() {
				http.PostForm("http://example.com", url.Values{})
			}
		`, `
			package main

			import (
				"net/http"
				"net/url"
			)

			func f() {
				http.DefaultClient.PostForm("http://example.com", url.Values{})
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
