/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindRandSeed(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindRandSeed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"math/rand"
				"time"
			)

			func f() {
				rand.Seed(time.Now().UnixNano())
			}
		`),
	)
}

func TestFindRandSeedNoChangeIntn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindRandSeed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "math/rand"

			func f() int {
				return rand.Intn(10)
			}
		`),
	)
}
