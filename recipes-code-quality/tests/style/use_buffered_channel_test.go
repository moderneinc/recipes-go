/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseBufferedChannel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseBufferedChannel{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				ch := make(chan int)
				_ = ch
			}
		`),
	)
}

func TestUseBufferedChannelNoChangeBuffered(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseBufferedChannel{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				ch := make(chan int, 10)
				_ = ch
			}
		`),
	)
}

func TestUseBufferedChannelNoChangeSlice(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseBufferedChannel{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				s := make([]int, 5)
				_ = s
			}
		`),
	)
}
