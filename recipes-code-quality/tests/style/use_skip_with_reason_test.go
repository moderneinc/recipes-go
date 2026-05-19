/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseSkipWithReason(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseSkipWithReason{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "testing"

			func TestFoo(t *testing.T) {
				t.Skip()
			}
		`, `
			package main

			import "testing"

			func TestFoo(t *testing.T) {
				t.Skip("TODO: add reason")
			}
		`),
	)
}

func TestUseSkipWithReasonNoChangeWithMessage(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseSkipWithReason{})
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
