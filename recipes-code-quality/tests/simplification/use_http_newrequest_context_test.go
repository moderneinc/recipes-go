/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseHttpNewRequestWithContext(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.UseHttpNewRequestWithContext{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net/http"

			func f() (*http.Request, error) {
				return http.NewRequest("GET", "https://example.com", nil)
			}
		`, `
			package main

			import "net/http"

			func f() (*http.Request, error) {
				return http.NewRequestWithContext(context.TODO(), "GET", "https://example.com", nil)
			}
		`),
	)
}
