/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidDotImportRemoved(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import . "fmt"

			func main() {
				Println("hello")
			}
		`, `
			package main

			import "fmt"

			func main() {
				Println("hello")
			}
		`),
	)
}

func TestAvoidDotImportNoChangeNormalImport(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func main() {
				fmt.Println("hello")
			}
		`),
	)
}

func TestAvoidDotImportNoChangeAliasedImport(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidDotImport{})
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
