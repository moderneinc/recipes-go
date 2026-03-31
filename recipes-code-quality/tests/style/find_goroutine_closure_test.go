/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindGoroutineClosure(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindGoroutineClosure{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				go func() {
					println("hi")
				}()
			}
		`),
	)
}

func TestFindGoroutineClosureNoChangeNamedFunc(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindGoroutineClosure{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func doWork() {}

			func f() {
				go doWork()
			}
		`),
	)
}
