/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestCheckTemplateExecuteError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.CheckTemplateExecuteError{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "html/template"

			func f(tmpl *template.Template, w *template.Template, data any) error {
				tmpl.Execute(w, data)
				return nil
			}
		`, `
			package main

			import "html/template"

			func f(tmpl *template.Template, w *template.Template, data any) error {
				if err := tmpl.Execute(w, data); err != nil {
					return err
				}
				return nil
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

			func f(tmpl *template.Template, w *template.Template, data any) error {
				tmpl.ExecuteTemplate(w, "page", data)
				return nil
			}
		`, `
			package main

			import "html/template"

			func f(tmpl *template.Template, w *template.Template, data any) error {
				if err := tmpl.ExecuteTemplate(w, "page", data); err != nil {
					return err
				}
				return nil
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

func TestCheckTemplateExecuteErrorNoChangeNoErrorReturn(t *testing.T) {
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
