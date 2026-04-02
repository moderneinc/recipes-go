/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindLockInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AvoidLockInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sync"

			func f(mu *sync.Mutex) {
				for i := 0; i < 10; i++ {
					mu.Lock()
					mu.Unlock()
				}
			}
		`),
	)
}

func TestFindRLockInRangeLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AvoidLockInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sync"

			func f(mu *sync.RWMutex, items []string) {
				for range items {
					mu.RLock()
					mu.RUnlock()
				}
			}
		`),
	)
}

func TestFindLockNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AvoidLockInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sync"

			func f(mu *sync.Mutex) {
				mu.Lock()
				mu.Unlock()
			}
		`),
	)
}
