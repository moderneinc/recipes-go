/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseCryptoRandIntn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseCryptoRand{})
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

func TestUseCryptoRandFloat64(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseCryptoRand{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "math/rand"

			func f() float64 {
				return rand.Float64()
			}
		`),
	)
}

func TestUseCryptoRandNoChangeOtherPkg(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseCryptoRand{})
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
