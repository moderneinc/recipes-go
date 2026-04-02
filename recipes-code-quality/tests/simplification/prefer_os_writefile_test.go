/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferOsWriteFile(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferOsWriteFile{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "io/ioutil"

			func f(name string, data []byte) error {
				return ioutil.WriteFile(name, data, 0644)
			}
		`, `
			package main

			import "io/ioutil"

			func f(name string, data []byte) error {
				return os.WriteFile(name, data, 0644)
			}
		`),
	)
}
