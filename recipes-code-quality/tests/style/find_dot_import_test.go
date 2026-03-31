/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindDotImport(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import . "fmt"

			func f() {
				Println("hello")
			}
		`),
	)
}

func TestFindDotImportNoChangeRegular(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDotImport{})
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

func TestFindDotImportNoChangeAlias(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import f "fmt"

			func main() {
				f.Println("hello")
			}
		`),
	)
}
