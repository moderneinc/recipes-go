/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindChannelClose(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindChannelClose{})
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

func TestFindChannelCloseBuffered(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindChannelClose{})
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

func TestFindChannelCloseNoChangeLenBuiltin(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindChannelClose{})
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

func TestFindChannelCloseNoChangeOtherFunc(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindChannelClose{})
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
