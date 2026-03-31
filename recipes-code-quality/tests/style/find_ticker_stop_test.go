/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindTickerWithoutStop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTickerWithoutStop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f() {
				time.NewTicker(time.Second)
			}
		`),
	)
}

func TestFindTickerWithoutStopNoChangeTimer(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTickerWithoutStop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f() {
				time.NewTimer(time.Second)
			}
		`),
	)
}
