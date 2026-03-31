/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindOsFileOpen(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindOsFileOpen{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				os.Open("file")
			}
		`),
	)
}

func TestFindOsFileOpenNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindOsFileOpen{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				os.Getenv("X")
			}
		`),
	)
}
