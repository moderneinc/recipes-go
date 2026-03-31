/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindHttpNewRequest(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHttpNewRequest{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() {
				http.NewRequest("GET", "http://example.com", nil)
			}
		`),
	)
}

func TestFindHttpNewRequestNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHttpNewRequest{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"context"
				"net/http"
			)

			func f() {
				ctx := context.Background()
				http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
			}
		`),
	)
}
