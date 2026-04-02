/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveDebugPrintFmtPrintln(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveDebugPrint{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() {
				fmt.Println("debug")
			}
		`, `
			package main

			import "fmt"

			func f() {
			}
		`),
	)
}

func TestRemoveDebugPrintBuiltinPrintln(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveDebugPrint{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				println("debug")
			}
		`, `
			package main

			func f() {
			}
		`),
	)
}

func TestRemoveDebugPrintNoChangeLogPrintln(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveDebugPrint{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "log"

			func f() {
				log.Println("info")
			}
		`),
	)
}
