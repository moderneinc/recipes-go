/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferOsMkdirTemp(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferOsMkdirTemp{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "io/ioutil"

			func f() (string, error) {
				return ioutil.TempDir("", "prefix")
			}
		`, `
			package main

			import "io/ioutil"

			func f() (string, error) {
				return os.MkdirTemp("", "prefix")
			}
		`),
	)
}
