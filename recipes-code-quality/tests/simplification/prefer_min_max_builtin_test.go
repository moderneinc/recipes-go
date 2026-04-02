/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferMinBuiltin(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferMinMaxBuiltin{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "math"

			func f(a, b float64) float64 {
				return math.Min(a, b)
			}
		`, `
			package main

			import "math"

			func f(a, b float64) float64 {
				return min(a, b)
			}
		`),
	)
}

func TestPreferMaxBuiltin(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferMinMaxBuiltin{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "math"

			func f(a, b float64) float64 {
				return math.Max(a, b)
			}
		`, `
			package main

			import "math"

			func f(a, b float64) float64 {
				return max(a, b)
			}
		`),
	)
}

func TestPreferMinMaxBuiltinNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferMinMaxBuiltin{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "math"

			func f(x float64) float64 {
				return math.Abs(x)
			}
		`),
	)
}
