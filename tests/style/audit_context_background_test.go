/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditContextBackground(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditContextBackground{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f() {
				ctx := context.Background()
				_ = ctx
			}
		`, `
			package main

			import "context"

			func f() {
				ctx := /*~~(context.Background() call; consider using a passed context instead)~~>*/context.Background()
				_ = ctx
			}
		`),
	)
}

func TestAuditContextBackgroundNoChangeTodo(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditContextBackground{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "context"

			func f() {
				ctx := context.TODO()
				_ = ctx
			}
		`),
	)
}

// A local named `context` with a Background method is not the context package.
func TestAuditContextBackgroundIgnoresShadowingLocal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditContextBackground{})
	spec.RewriteRun(t, test.Golang(`
		package main

		type fake struct{}

		func (fake) Background() {}

		func f() {
			context := fake{}
			context.Background()
		}
	`))
}
