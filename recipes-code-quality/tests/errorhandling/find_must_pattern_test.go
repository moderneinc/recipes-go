/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindMustFunction(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindMustFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "text/template"

			func f() *template.Template {
				return template.Must(template.New("t").Parse(""))
			}
		`),
	)
}

func TestFindMustFunctionNoChangeNew(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindMustFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "text/template"

			func f() *template.Template {
				return template.New("t")
			}
		`),
	)
}
