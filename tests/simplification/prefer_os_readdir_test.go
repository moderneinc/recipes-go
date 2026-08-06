/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferOsReadDir(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferOsReadDir{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "io/ioutil"

			func f(name string) {
				entries, _ := ioutil.ReadDir(name)
				_ = entries
			}
		`, `
			package main

			import (
				"os"
			)

			func f(name string) {
				entries, _ := os.ReadDir(name)
				_ = entries
			}
		`),
	)
}

func TestPreferOsReadDirNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferOsReadDir{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(name string) {
				entries, _ := os.ReadDir(name)
				_ = entries
			}
		`),
	)
}

// Skips a direct return of []os.FileInfo, where os.ReadDir's []os.DirEntry would not compile.
func TestPreferOsReadDirNoChangeFileInfoContext(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferOsReadDir{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"io/ioutil"
				"os"
			)

			func f(name string) ([]os.FileInfo, error) {
				return ioutil.ReadDir(name)
			}
		`),
	)
}
