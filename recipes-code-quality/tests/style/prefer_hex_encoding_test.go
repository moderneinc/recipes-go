/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferHexEncoding(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferHexEncoding{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(data []byte) string {
				return fmt.Sprintf("%x", data)
			}
		`, `
			package main

			import "fmt"

			func f(data []byte) string {
				return hex.EncodeToString(data)
			}
		`),
	)
}

func TestPreferHexEncodingNoChangeOtherVerb(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferHexEncoding{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(data []byte) string {
				return fmt.Sprintf("%s", data)
			}
		`),
	)
}
