/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindInitFunction(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindInitFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func init() {
				println("startup")
			}
		`),
	)
}

func TestFindInitFunctionNoChangeRegularFunc(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindInitFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {
				println("hello")
			}
		`),
	)
}

func TestFindInitFunctionNoChangeMethodReceiver(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindInitFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type Foo struct{}

			func (f *Foo) init() {
			}
		`),
	)
}
