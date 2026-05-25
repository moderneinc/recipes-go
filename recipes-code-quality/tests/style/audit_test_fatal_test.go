/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditTestFatal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditTestFatal{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "testing"

			func TestFoo(t *testing.T) {
				t.Fatal("fail")
			}
		`, `
			package main

			import "testing"

			func TestFoo(t *testing.T) {/*~~(t.Fatal call found; consider t.Error in goroutines)~~>*/
				t.Fatal("fail")
			}
		`),
	)
}

func TestAuditTestFatalf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditTestFatal{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "testing"

			func TestFoo(t *testing.T) {
				t.Fatalf("got %d", 1)
			}
		`, `
			package main

			import "testing"

			func TestFoo(t *testing.T) {/*~~(t.Fatal call found; consider t.Error in goroutines)~~>*/
				t.Fatalf("got %d", 1)
			}
		`),
	)
}

func TestAuditTestFatalNoChangeError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditTestFatal{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "testing"

			func TestFoo(t *testing.T) {
				t.Error("fail")
			}
		`),
	)
}
