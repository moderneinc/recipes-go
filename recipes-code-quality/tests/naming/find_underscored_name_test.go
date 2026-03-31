/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindUnderscoredExportedName(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindUnderscoredExportedName{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func Get_User() {
			}
		`),
	)
}

func TestFindUnderscoredExportedNameNoChangeCamelCase(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindUnderscoredExportedName{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func GetUser() {
			}
		`),
	)
}

func TestFindUnderscoredExportedNameNoChangeUnexported(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindUnderscoredExportedName{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func get_user() {
			}
		`),
	)
}
