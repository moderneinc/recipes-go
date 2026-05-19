/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferStringsBuilderWriteString(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsBuilderWriteString{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"fmt"
				"strings"
			)

			func f(s string) string {
				var b strings.Builder
				fmt.Fprintf(&b, "%s", s)
				return b.String()
			}
		`, `
			package main

			import (
				"fmt"
				"strings"
			)

			func f(s string) string {
				var b strings.Builder
				b.WriteString(s)
				return b.String()
			}
		`),
	)
}

func TestPreferStringsBuilderWriteStringNoChangeFormat(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsBuilderWriteString{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"fmt"
				"strings"
			)

			func f(x int) string {
				var b strings.Builder
				fmt.Fprintf(&b, "%d", x)
				return b.String()
			}
		`),
	)
}
