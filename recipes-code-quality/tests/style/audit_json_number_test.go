/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditJsonNumber(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditJsonNumber{})
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

func TestAuditJsonNumberNoChangeDecoder(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditJsonNumber{})
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
