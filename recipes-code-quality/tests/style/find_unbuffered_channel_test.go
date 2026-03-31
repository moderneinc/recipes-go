/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindUnbufferedChannel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindUnbufferedChannel{})
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

func TestFindUnbufferedChannelNoChangeBuffered(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindUnbufferedChannel{})
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

func TestFindUnbufferedChannelNoChangeSlice(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindUnbufferedChannel{})
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
