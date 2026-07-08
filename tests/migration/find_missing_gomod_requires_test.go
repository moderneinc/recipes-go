/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindMissingGoModRequiresFlagsUncoveredImport(t *testing.T) {
	// given
	spec := test.NewRecipeSpec().WithRecipe(&migration.FindMissingGoModRequires{})

	// when / then
	spec.RewriteRun(t,
		test.GoProject("example",
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/foo/bar v1.2.3
			`),
			test.Golang(`
				package main

				import (
					"fmt"

					"github.com/foo/bar"
					"github.com/baz/qux"
				)

				func main() {
					fmt.Println(bar.A, qux.B)
				}
			`, `
				package main

				import (
					"fmt"

					"github.com/foo/bar"
					/*~~(missing go.mod requirement)~~>*/"github.com/baz/qux"
				)

				func main() {
					fmt.Println(bar.A, qux.B)
				}
			`),
		),
	)
}

func TestFindMissingGoModRequiresNoChangeWhenAllCovered(t *testing.T) {
	// given
	spec := test.NewRecipeSpec().WithRecipe(&migration.FindMissingGoModRequires{})

	// when / then
	spec.RewriteRun(t,
		test.GoProject("example",
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/foo/bar v1.2.3
			`),
			test.Golang(`
				package main

				import (
					"fmt"
					"strings"

					"github.com/foo/bar/sub"
					"example.com/app/internal"
				)

				func main() {
					fmt.Println(strings.TrimSpace(bar.A), sub.C, internal.D)
				}
			`),
		),
	)
}
