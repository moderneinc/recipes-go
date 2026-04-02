/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindEmptyFmtSprintfEmpty(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptyFmtSprintf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() string {
				return fmt.Sprintf("")
			}
		`, `
			package main

			import "fmt"

			func f() string {
				return ""
			}
		`),
	)
}

func TestFindEmptyFmtSprintfNoChangeNonEmpty(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptyFmtSprintf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() string {
				return fmt.Sprintf("hello")
			}
		`),
	)
}

func TestFindEmptyFmtSprintfNoChangeWithArgs(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindEmptyFmtSprintf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(name string) string {
				return fmt.Sprintf("hello %s", name)
			}
		`),
	)
}
