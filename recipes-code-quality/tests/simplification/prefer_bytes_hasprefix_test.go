/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferBytesHasPrefixPositive(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesHasPrefix{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b, prefix []byte) bool {
				return bytes.Index(b, prefix) == 0
			}
		`, `
			package main

			import "bytes"

			func f(b, prefix []byte) bool {
				return bytes.HasPrefix(b, prefix)
			}
		`),
	)
}

func TestPreferBytesHasPrefixNegative(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesHasPrefix{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b, prefix []byte) bool {
				return bytes.Index(b, prefix) != 0
			}
		`, `
			package main

			import "bytes"

			func f(b, prefix []byte) bool {
				return !bytes.HasPrefix(b, prefix)
			}
		`),
	)
}

func TestPreferBytesHasPrefixNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferBytesHasPrefix{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b, prefix []byte) int {
				return bytes.Index(b, prefix)
			}
		`),
	)
}
