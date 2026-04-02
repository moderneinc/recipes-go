/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestReduceNestingDepth(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceNestingDepth{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				for i := 0; i < 10; i++ {
					if true {
						if true {
							if true {
								x := 1
								_ = x
							}
						}
					}
				}
			}
		`),
	)
}

func TestReduceNestingDepthNoChangeShallow(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.ReduceNestingDepth{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				if true {
					x := 1
					_ = x
				}
			}
		`),
	)
}
