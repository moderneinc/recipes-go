/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsureTimerStopped(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTimerStopped{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f() {
				timer := time.NewTimer(time.Second)
				_ = timer
			}
		`, `
			package main

			import "time"

			func f() {
				timer := time.NewTimer(time.Second)
				defer timer.Stop()
				_ = timer
			}
		`),
	)
}

// A time.AfterFunc callback is meant to run after the enclosing function
// returns, which is exactly when a deferred Stop would cancel it.
func TestEnsureTimerStoppedNoChangeAfterFunc(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTimerStopped{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func schedule(f func()) {
				t := time.AfterFunc(time.Minute, f)
				_ = t
			}
		`),
	)
}

func TestEnsureTimerStoppedNoChangeNow(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTimerStopped{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f() {
				time.Now()
			}
		`),
	)
}

func TestEnsureTimerStoppedAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureTimerStopped{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f() {
				timer := time.NewTimer(time.Second)
				defer timer.Stop()
			}
		`),
	)
}
