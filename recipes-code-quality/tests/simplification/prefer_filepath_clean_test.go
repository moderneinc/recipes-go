/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferFilepathClean(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferFilepathClean{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "path/filepath"

			func f(p string) string {
				return filepath.Join(filepath.Clean(p))
			}
		`, `
			package main

			import "path/filepath"

			func f(p string) string {
				return filepath.Clean(p)
			}
		`),
	)
}

func TestPreferFilepathCleanNoChangeMultipleArgs(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferFilepathClean{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "path/filepath"

			func f(a, b string) string {
				return filepath.Join(a, b)
			}
		`),
	)
}
