/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveDeprecatedRandSeed(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveDeprecatedRandSeed{})
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
		`, `
			package main

			import (
				"math/rand"
				"time"
			)

			func f() {
			}
		`),
	)
}

func TestRemoveDeprecatedRandSeedNoChangeIntn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.RemoveDeprecatedRandSeed{})
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
