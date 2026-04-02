/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestCheckTemplateExecuteError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.CheckTemplateExecuteError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "html/template"

			func f(tmpl *template.Template, w *template.Template, data any) {
				tmpl.Execute(w, data)
			}
		`),
	)
}

func TestCheckTemplateExecuteErrorTemplate(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.CheckTemplateExecuteError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "html/template"

			func f(tmpl *template.Template, w *template.Template, data any) {
				tmpl.ExecuteTemplate(w, "page", data)
			}
		`),
	)
}

func TestCheckTemplateExecuteErrorNoChangeName(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.CheckTemplateExecuteError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "html/template"

			func f(tmpl *template.Template) {
				tmpl.Name()
			}
		`),
	)
}
