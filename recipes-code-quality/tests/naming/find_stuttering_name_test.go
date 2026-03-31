/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindStutteringName(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindStutteringName{})
	spec.RewriteRun(t,
		test.Golang(`
			package http

			func HttpGet() {
			}
		`),
	)
}

func TestFindStutteringNameNoChangeShort(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindStutteringName{})
	spec.RewriteRun(t,
		test.Golang(`
			package http

			func Get() {
			}
		`),
	)
}

func TestFindStutteringNameNoChangeUnexported(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindStutteringName{})
	spec.RewriteRun(t,
		test.Golang(`
			package http

			func httpGet() {
			}
		`),
	)
}
