/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
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

func TestExportedMethodNoComment(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AddExportedFuncComment{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type Service struct{}

			func (s *Service) Run() {
			}
		`, `
			package main

			type Service struct{}

			// Run ...
			func (s *Service) Run() {
			}
		`),
	)
}

func TestExportedMethodWithDocComment(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AddExportedFuncComment{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type Service struct{}

			// Run does something
			func (s *Service) Run() {
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

func TestExportedFuncWithMultiLineDocComment(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AddExportedFuncComment{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			// Hello does something.
			//
			// It has more to say about it.
			func Hello() {
			}
		`),
	)
}

func TestCommentDetachedByBlankLineIsNotADocComment(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AddExportedFuncComment{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			// Hello used to be declared here.

			func Hello() {
			}
		`, `
			package main

			// Hello used to be declared here.

			// Hello ...
			func Hello() {
			}
		`),
	)
}
