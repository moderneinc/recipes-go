/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestKeepInterfacesSmall(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.KeepInterfacesSmall{})
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

func TestKeepInterfacesSmallNoChangeSmall(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.KeepInterfacesSmall{})
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
