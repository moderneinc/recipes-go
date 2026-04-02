/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferIoWriteString(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferIoWriteString{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"fmt"
				"os"
			)

			func f(s string) {
				fmt.Fprintf(os.Stdout, "%s", s)
			}
		`, `
			package main

			import (
				"fmt"
				"os"
			)

			func f(s string) {
				io.WriteString(os.Stdout, s)
			}
		`),
	)
}

func TestPreferIoWriteStringNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferIoWriteString{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"fmt"
				"os"
			)

			func f(x int) {
				fmt.Fprintf(os.Stdout, "%d", x)
			}
		`),
	)
}
