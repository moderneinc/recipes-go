/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferOsReadDir(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferOsReadDir{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "io/ioutil"

			func f(name string) ([]os.FileInfo, error) {
				return ioutil.ReadDir(name)
			}
		`, `
			package main

			import "io/ioutil"

			func f(name string) ([]os.FileInfo, error) {
				return os.ReadDir(name)
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
