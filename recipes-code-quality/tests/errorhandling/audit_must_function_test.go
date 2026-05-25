/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditMustFunction(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AuditMustFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "text/template"

			func f() *template.Template {
				return template.Must(template.New("t").Parse(""))
			}
		`, `
			package main

			import "text/template"

			func f() *template.Template {
				return/*~~(Must* function panics on error; use with care)~~>*/ template.Must(template.New("t").Parse(""))
			}
		`),
	)
}

func TestAuditMustFunctionNoChangeNew(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AuditMustFunction{})
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
