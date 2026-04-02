/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidEmptyInterfaceParam(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidEmptyInterfaceParam{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x interface{}) {
			}
		`, `
			package main

			func f(x any) {
			}
		`),
	)
}

func TestAvoidEmptyInterfaceParamAny(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidEmptyInterfaceParam{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x any) {
			}
		`),
	)
}

func TestAvoidEmptyInterfaceParamNoChangeConcreteType(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidEmptyInterfaceParam{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
			}
		`),
	)
}
