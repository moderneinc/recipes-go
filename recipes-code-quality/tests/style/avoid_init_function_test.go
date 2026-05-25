/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidInitFunction(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidInitFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func init() {
				println("startup")
			}
		`, `
			package main

			func /*~~(consider removing init function)~~>*/init() {
				println("startup")
			}
		`),
	)
}

func TestAvoidInitFunctionNoChangeRegularFunc(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidInitFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {
				println("hello")
			}
		`),
	)
}

func TestAvoidInitFunctionNoChangeMethodReceiver(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidInitFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type Foo struct{}

			func (f *Foo) init() {
			}
		`),
	)
}
