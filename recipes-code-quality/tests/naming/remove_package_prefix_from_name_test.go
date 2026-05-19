/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemovePackagePrefixFromName(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.RemovePackagePrefixFromName{})
	spec.RewriteRun(t,
		test.Golang(`
			package http

			func HttpGet() {
			}
		`, `
			package http

			func Get() {
			}
		`),
	)
}

func TestRemovePackagePrefixFromNameNoChangeShort(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.RemovePackagePrefixFromName{})
	spec.RewriteRun(t,
		test.Golang(`
			package http

			func Get() {
			}
		`),
	)
}

func TestRemovePackagePrefixFromNameNoChangeUnexported(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.RemovePackagePrefixFromName{})
	spec.RewriteRun(t,
		test.Golang(`
			package http

			func httpGet() {
			}
		`),
	)
}
