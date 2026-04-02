/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditTestMain(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditTestMain{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"os"
				"testing"
			)

			func TestMain(m *testing.M) {
				os.Exit(m.Run())
			}
		`),
	)
}

func TestAuditTestMainNoChangeRegularTest(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditTestMain{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "testing"

			func TestFoo(t *testing.T) {
				t.Log("hello")
			}
		`),
	)
}
