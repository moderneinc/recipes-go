/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditChannelClose(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditChannelClose{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				ch := make(chan int)
				close(ch)
			}
		`),
	)
}

func TestAuditChannelCloseBuffered(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditChannelClose{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				ch := make(chan string, 10)
				close(ch)
			}
		`),
	)
}

func TestAuditChannelCloseNoChangeLenBuiltin(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditChannelClose{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				ch := make(chan int, 5)
				_ = len(ch)
			}
		`),
	)
}

func TestAuditChannelCloseNoChangeOtherFunc(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditChannelClose{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() {
				fmt.Println("hello")
			}
		`),
	)
}
