/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindCopyInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.OptimizeCopyInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				dst := make([]byte, 10)
				src := []byte("hello")
				for i := 0; i < 10; i++ {
					copy(dst, src)
				}
			}
		`),
	)
}

func TestFindCopyNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.OptimizeCopyInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				dst := make([]byte, 10)
				src := []byte("hello")
				copy(dst, src)
			}
		`),
	)
}
