/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindChannelLenCheckEqualZero(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindChannelLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ch chan int) bool {
				return len(ch) == 0
			}
		`),
	)
}

func TestFindChannelLenCheckGreaterZero(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindChannelLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ch chan int) bool {
				return len(ch) > 0
			}
		`),
	)
}

func TestFindChannelLenCheckNoChangeSlice(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.FindChannelLenCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s []int) bool {
				return len(s) == 0
			}
		`),
	)
}
