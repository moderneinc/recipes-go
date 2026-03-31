/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindEmptyInterfaceParam(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindEmptyInterfaceParam{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
			}
		`),
	)
}

func TestFindEmptyInterfaceParamAny(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindEmptyInterfaceParam{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x any) {
			}
		`),
	)
}

func TestFindEmptyInterfaceParamNoChangeConcreteType(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindEmptyInterfaceParam{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
			}
		`),
	)
}
