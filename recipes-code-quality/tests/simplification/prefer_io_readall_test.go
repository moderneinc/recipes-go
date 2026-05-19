/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferIoReadAll(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferIoReadAll{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "io/ioutil"

			func f(r *Reader) ([]byte, error) {
				return ioutil.ReadAll(r)
			}
		`, `
			package main

			import "io/ioutil"

			func f(r *Reader) ([]byte, error) {
				return io.ReadAll(r)
			}
		`),
	)
}
