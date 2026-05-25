/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidChannelLenCheckEqualZero(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.AvoidChannelLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ch chan int) bool {
				return len(ch) == 0
			}
		`, `
			package main

			func f(ch chan int) bool {
				return/*~~(channel length check is racy; the value can change between check and send/receive)~~>*/ len(ch) == 0
			}
		`),
	)
}

func TestAvoidChannelLenCheckGreaterZero(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.AvoidChannelLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ch chan int) bool {
				return len(ch) > 0
			}
		`, `
			package main

			func f(ch chan int) bool {
				return/*~~(channel length check is racy; the value can change between check and send/receive)~~>*/ len(ch) > 0
			}
		`),
	)
}

func TestAvoidChannelLenCheckNoChangeSlice(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.AvoidChannelLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) bool {
				return len(s) == 0
			}
		`),
	)
}
