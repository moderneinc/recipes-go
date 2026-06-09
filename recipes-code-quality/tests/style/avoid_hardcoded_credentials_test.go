/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidHardcodedCredentialsPassword(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidHardcodedCredentials{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var password = "hunter2"
		`, `
			package main

			import "os"

			var password = os.Getenv("PASSWORD")
		`),
	)
}

func TestAvoidHardcodedCredentialsDbPassword(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidHardcodedCredentials{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var dbPassword = "secret123"
		`, `
			package main

			import "os"

			var dbPassword = os.Getenv("DB_PASSWORD")
		`),
	)
}

func TestAvoidHardcodedCredentialsNoChangeNonCredential(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidHardcodedCredentials{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var name = "hello"
		`),
	)
}

func TestAvoidHardcodedCredentialsNoChangeNonString(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidHardcodedCredentials{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			var password = 42
		`),
	)
}
