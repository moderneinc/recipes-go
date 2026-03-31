/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindStdLogPrintln(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindStdLog{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "log"

			func f() {
				log.Println("hello")
			}
		`),
	)
}

func TestFindStdLogPrintf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindStdLog{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "log"

			func f() {
				log.Printf("hello %s", "world")
			}
		`),
	)
}

func TestFindStdLogFatal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindStdLog{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "log"

			func main() {
				log.Fatal("error")
			}
		`),
	)
}

func TestFindStdLogNoChangeFmt(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindStdLog{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() {
				fmt.Println("hello")
			}
		`),
	)
}
