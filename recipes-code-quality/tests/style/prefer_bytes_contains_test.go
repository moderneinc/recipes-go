/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferBytesContainsNotEqualNegOne(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferBytesContains{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b, sub []byte) bool {
				return bytes.Index(b, sub) != -1
			}
		`, `
			package main

			import "bytes"

			func f(b, sub []byte) bool {
				return bytes.Contains(b, sub)
			}
		`),
	)
}

func TestPreferBytesContainsNoChangeGreaterThanZero(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferBytesContains{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b, sub []byte) bool {
				return bytes.Index(b, sub) > 0
			}
		`),
	)
}
