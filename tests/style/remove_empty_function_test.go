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

func TestRemoveEmptyFunctionNoChangeForMain(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {
			}
		`),
	)
}

func TestRemoveEmptyFunctionRemovesMainOutsideMainPackage(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package helper

			func main() {
			}
		`, `
			package helper
		`),
	)
}

func TestRemoveEmptyFunctionNoChangeForInit(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package helper

			func init() {
			}
		`),
	)
}

func TestRemoveEmptyFunctionNoChangeForExported(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package helper

			func Noop() {}
		`),
	)
}

func TestRemoveEmptyFunctionNoChangeForInterfaceImplementation(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyFunction{})
	spec.RewriteRun(t,
		test.Golang(`
			package helper

			type Logger interface {
				Log(msg string)
			}

			type noopLogger struct{}

			func (noopLogger) Log(msg string) {}
		`),
	)
}

// The chain that reaches urfave/cli's hello-world example: RemoveDebugPrint
// hands RemoveEmptyFunction an already-emptied `main`.
func TestRemoveEmptyFunctionKeepsMainAfterDebugPrintRemoval(t *testing.T) {
	emptiedMain := `
			// example hello world used for binary size checking

			package main

			func main() {
			}
		`
	test.NewRecipeSpec().WithRecipe(&style.RemoveDebugPrint{}).RewriteRun(t,
		test.Golang(`
			// example hello world used for binary size checking

			package main

			import "fmt"

			func main() {
				fmt.Println("hello world")
			}
		`, emptiedMain),
	)
	test.NewRecipeSpec().WithRecipe(&style.RemoveEmptyFunction{}).RewriteRun(t,
		test.Golang(emptiedMain),
	)
}
