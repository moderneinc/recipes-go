/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferStrconvAtoi(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStrconvAtoi{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strconv"

			func f(s string) (int64, error) {
				return strconv.ParseInt(s, 10, 0)
			}
		`, `
			package main

			import "strconv"

			func f(s string) (int64, error) {
				return strconv.Atoi(s)
			}
		`),
	)
}

func TestPreferStrconvAtoiNoChangeBase16(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStrconvAtoi{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strconv"

			func f(s string) (int64, error) {
				return strconv.ParseInt(s, 16, 0)
			}
		`),
	)
}

func TestPreferStrconvAtoiNoChangeBitSize(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStrconvAtoi{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strconv"

			func f(s string) (int64, error) {
				return strconv.ParseInt(s, 10, 64)
			}
		`),
	)
}
