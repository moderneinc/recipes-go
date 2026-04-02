/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseMixedCaps(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseMixedCaps{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func Get_User() {
			}
		`, `
			package main

			func GetUser() {
			}
		`),
	)
}

func TestUseMixedCapsNoChangeCamelCase(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseMixedCaps{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func GetUser() {
			}
		`),
	)
}

func TestUseMixedCapsNoChangeUnexported(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseMixedCaps{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func get_user() {
			}
		`),
	)
}
