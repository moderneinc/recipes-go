/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestReplaceTimeUntilSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.ReplaceTimeUntilWithUntil{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f(deadline time.Time) time.Duration {
				return deadline.Sub(time.Now())
			}
		`, `
			package main

			import "time"

			func f(deadline time.Time) time.Duration {
				return time.Until(deadline)
			}
		`),
	)
}

func TestReplaceTimeUntilNoChangeRegularSub(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.ReplaceTimeUntilWithUntil{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f(a, b time.Time) time.Duration {
				return a.Sub(b)
			}
		`),
	)
}
