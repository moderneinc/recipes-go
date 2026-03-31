/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindLargeInterface(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindLargeInterface{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type Large interface {
				A()
				B()
				C()
				D()
				E()
				F()
			}
		`),
	)
}

func TestFindLargeInterfaceNoChangeSmall(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindLargeInterface{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type Small interface {
				A()
				B()
			}
		`),
	)
}
