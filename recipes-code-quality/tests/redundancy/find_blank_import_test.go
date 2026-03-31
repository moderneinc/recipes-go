/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindBlankImportFound(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindBlankImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import _ "net/http/pprof"

			func f() {}
		`),
	)
}

func TestFindBlankImportNoChangeRegular(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindBlankImport{})
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

func TestFindBlankImportNoChangeAlias(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindBlankImport{})
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
