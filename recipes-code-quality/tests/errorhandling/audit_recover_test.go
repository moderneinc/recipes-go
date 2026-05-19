/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditRecover(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AuditRecover{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {
				defer func() {
					recover()
				}()
			}
		`),
	)
}

func TestAuditRecoverNoChangePanic(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AuditRecover{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func main() {
				panic("x")
			}
		`),
	)
}
