/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsureTickerStopped(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTickerStopped{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f() {
				ticker := time.NewTicker(time.Second)
				_ = ticker
			}
		`, `
			package main

			import "time"

			func f() {
				ticker := time.NewTicker(time.Second)
				defer ticker.Stop()
				_ = ticker
			}
		`),
	)
}

func TestEnsureTickerStoppedNoChangeTimer(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTickerStopped{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f() {
				time.NewTimer(time.Second)
			}
		`),
	)
}

func TestEnsureTickerStoppedAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTickerStopped{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f() {
				ticker := time.NewTicker(time.Second)
				defer ticker.Stop()
			}
		`),
	)
}
