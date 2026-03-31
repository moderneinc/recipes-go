/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindLabel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindLabel{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
			loop:
				for {
					break loop
				}
			}
		`),
	)
}

func TestFindLabelNoChangeWithoutLabel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindLabel{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				for {
					break
				}
			}
		`),
	)
}
