/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseErrorMethod(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UseErrorMethod{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(err error) string {
				return fmt.Sprint(err)
			}
		`, `
			package main

			import "fmt"

			func f(err error) string {
				return err.Error()
			}
		`),
	)
}

func TestUseErrorMethodNoChangeInt(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UseErrorMethod{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() string {
				return fmt.Sprint(42)
			}
		`),
	)
}
