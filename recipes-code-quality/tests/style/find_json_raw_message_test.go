/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindJsonRawMessage(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindJsonRawMessage{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func f() {
				var raw json.RawMessage
				_ = raw
			}
		`),
	)
}

func TestFindJsonRawMessageNoChangeMarshal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindJsonRawMessage{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func f() {
				x := map[string]string{"key": "value"}
				_, _ = json.Marshal(x)
			}
		`),
	)
}
