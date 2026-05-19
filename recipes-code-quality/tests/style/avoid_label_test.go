/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidLabel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidLabel{})
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

func TestAvoidLabelNoChangeWithoutLabel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidLabel{})
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
