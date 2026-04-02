/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestReplaceTimeSinceSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.ReplaceTimeSinceWithSince{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f(t time.Time) time.Duration {
				return time.Now().Sub(t)
			}
		`, `
			package main

			import "time"

			func f(t time.Time) time.Duration {
				return time.Since(t)
			}
		`),
	)
}

func TestReplaceTimeSinceNoChangeDirectSub(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.ReplaceTimeSinceWithSince{})
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
