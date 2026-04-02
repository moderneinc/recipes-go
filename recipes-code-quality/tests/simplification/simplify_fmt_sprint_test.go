/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyFmtSprintfV(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyFmtSprintf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(x int) string {
				return fmt.Sprintf("%v", x)
			}
		`, `
			package main

			import "fmt"

			func f(x int) string {
				return fmt.Sprint(x)
			}
		`),
	)
}

func TestSimplifyFmtSprintfNoChangeD(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyFmtSprintf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(x int) string {
				return fmt.Sprintf("%d", x)
			}
		`),
	)
}
