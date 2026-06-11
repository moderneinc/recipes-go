/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindHttpGetInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.BatchHttpCalls{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				for i := 0; i < 10; i++ {
					_, _ = http.Get("http://example.com")
				}
			}
		`, `
			package main

			import "net/http"

			func f() {
				for i := 0; i < 10; i++ {
					_, _ = /*~~(HTTP call in loop; making HTTP requests in tight loops can be slow)~~>*/http.Get("http://example.com")
				}
			}
		`),
	)
}

func TestFindHttpGetNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.BatchHttpCalls{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				_, _ = http.Get("http://example.com")
			}
		`),
	)
}
