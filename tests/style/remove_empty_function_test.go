/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveEmptyFunction(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {}
		`, `
			package main
		`),
	)
}

func TestRemoveEmptyFunctionNoChangeWithBody(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				return
			}
		`),
	)
}

func TestRemoveEmptyFunctionNoChangeWithReceiver(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type S struct{}

			func (s *S) Noop() {}
		`),
	)
}

func TestRemoveEmptyFunctionNoChangeWithReturnType(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func zero() int { return 0 }
		`),
	)
}

func TestRemoveEmptyFunctionNoChangeWithFunctionLiteral(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func cleanup() (string, func()) {
				return "", func() {}
			}
		`),
	)
}

func TestRemoveEmptyFunctionNoChangeWithFunctionLiteralArgument(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func run(fn func()) {
				fn()
			}

			func main() {
				run(func() {})
			}
		`),
	)
}
