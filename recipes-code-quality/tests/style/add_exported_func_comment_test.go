/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestExportedFuncNoComment(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AddExportedFuncComment{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func Hello() {
			}
		`, `
			package main

			// Hello ...
			func Hello() {
			}
		`),
	)
}

func TestUnexportedFuncNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AddExportedFuncComment{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func hello() {
			}
		`),
	)
}

func TestExportedFuncWithDocComment(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AddExportedFuncComment{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			// Hello does something
			func Hello() {
			}
		`),
	)
}
