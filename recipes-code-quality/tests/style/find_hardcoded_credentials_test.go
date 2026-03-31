/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindHardcodedCredentials(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHardcodedCredentials{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var password = "abc123"
		`),
	)
}

func TestFindHardcodedCredentialsNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindHardcodedCredentials{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var name = "alice"
		`),
	)
}
