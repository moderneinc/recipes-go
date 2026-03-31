/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindPermission777MkdirAll(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindPermission777{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				os.MkdirAll("/tmp/dir", 0777)
			}
		`),
	)
}

func TestFindPermission777NoChange0755(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindPermission777{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				os.MkdirAll("/tmp/dir", 0755)
			}
		`),
	)
}

func TestFindPermission777Chmod(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindPermission777{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				os.Chmod("/tmp/file", 0777)
			}
		`),
	)
}
