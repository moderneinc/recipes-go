/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferOsCreateTemp(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferOsCreateTemp{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "io/ioutil"

			func f() (*File, error) {
				return ioutil.TempFile("", "prefix")
			}
		`, `
			package main

			import "io/ioutil"

			func f() (*File, error) {
				return os.CreateTemp("", "prefix")
			}
		`),
	)
}
