/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseDocumentedBlankImportFound(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.UseDocumentedBlankImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import _ "net/http/pprof"

			func f() {}
		`, `
			package main

			import /*~~(blank import used for side effects)~~>*/_ "net/http/pprof"

			func f() {}
		`),
	)
}

func TestUseDocumentedBlankImportNoChangeRegular(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.UseDocumentedBlankImport{})
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

func TestUseDocumentedBlankImportNoChangeAlias(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.UseDocumentedBlankImport{})
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
