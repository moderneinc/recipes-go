/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindDebugPrintFmtPrintln(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDebugPrint{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() {
				fmt.Println("debug")
			}
		`),
	)
}

func TestFindDebugPrintBuiltinPrintln(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDebugPrint{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				println("debug")
			}
		`),
	)
}

func TestFindDebugPrintNoChangeLogPrintln(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDebugPrint{})
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
