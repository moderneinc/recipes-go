/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditGoroutineClosure(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditGoroutineClosure{})
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

func TestAuditGoroutineClosureNoChangeNamedFunc(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditGoroutineClosure{})
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
