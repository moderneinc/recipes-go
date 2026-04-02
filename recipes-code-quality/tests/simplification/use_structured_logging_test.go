/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseStructuredLoggingPrintln(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.UseStructuredLogging{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "log"

			func f() {
				log.Println("hello")
			}
		`, `
			package main

			import "log"

			func f() {
				slog.Info("hello")
			}
		`),
	)
}

func TestUseStructuredLoggingPrintf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.UseStructuredLogging{})
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

func TestUseStructuredLoggingFatal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.UseStructuredLogging{})
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

func TestUseStructuredLoggingNoChangeFmt(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.UseStructuredLogging{})
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
