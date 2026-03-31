/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindJsonNumber(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindJsonNumber{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func f() {
				var n json.Number
				_ = n
			}
		`),
	)
}

func TestFindJsonNumberNoChangeDecoder(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindJsonNumber{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"strings"
			)

			func f() {
				dec := json.NewDecoder(strings.NewReader("{}"))
				_ = dec
			}
		`),
	)
}
