/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindSkipWithoutReason(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSkipWithoutReason{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "testing"

			func TestFoo(t *testing.T) {
				t.Skip()
			}
		`),
	)
}

func TestFindSkipWithoutReasonNoChangeWithMessage(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindSkipWithoutReason{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "testing"

			func TestFoo(t *testing.T) {
				t.Skip("reason")
			}
		`),
	)
}
