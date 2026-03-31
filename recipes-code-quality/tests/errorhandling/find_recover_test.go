/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindRecover(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindRecover{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {
				defer func() {
					recover()
				}()
			}
		`),
	)
}

func TestFindRecoverNoChangePanic(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FindRecover{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {
				panic("x")
			}
		`),
	)
}
