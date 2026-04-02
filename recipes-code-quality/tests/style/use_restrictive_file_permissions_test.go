/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseRestrictiveFilePermissionsMkdirAll(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseRestrictiveFilePermissions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				os.MkdirAll("/tmp/dir", 0777)
			}
		`, `
			package main

			import "os"

			func f() {
				os.MkdirAll("/tmp/dir", 0755)
			}
		`),
	)
}

func TestUseRestrictiveFilePermissionsNoChange0755(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseRestrictiveFilePermissions{})
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

func TestUseRestrictiveFilePermissionsChmod(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseRestrictiveFilePermissions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				os.Chmod("/tmp/file", 0777)
			}
		`, `
			package main

			import "os"

			func f() {
				os.Chmod("/tmp/file", 0755)
			}
		`),
	)
}
