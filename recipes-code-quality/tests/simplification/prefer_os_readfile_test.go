/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferOsReadFile(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferOsReadFile{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "io/ioutil"

			func f(name string) ([]byte, error) {
				return ioutil.ReadFile(name)
			}
		`, `
			package main

			import "io/ioutil"

			func f(name string) ([]byte, error) {
				return os.ReadFile(name)
			}
		`),
	)
}
