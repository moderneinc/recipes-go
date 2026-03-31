/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindErrorFmtSprint(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindErrorFmtSprint{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(err error) string {
				return fmt.Sprint(err)
			}
		`),
	)
}

func TestFindErrorFmtSprintNoChangeInt(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindErrorFmtSprint{})
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
