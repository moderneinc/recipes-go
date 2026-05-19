/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsureHttpBodyClosedGet(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				resp, err := http.Get("http://example.com")
				_ = err
				_ = resp
			}
		`, `
			package main

			import "net/http"

			func f() {
				resp, err := http.Get("http://example.com")
				defer resp.Body.Close()
				_ = err
				_ = resp
			}
		`),
	)
}

func TestEnsureHttpBodyClosedNoChangeError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
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

func TestEnsureHttpBodyClosedAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureHttpBodyClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				resp, err := http.Get("http://example.com")
				defer resp.Body.Close()
				_ = err
			}
		`),
	)
}
