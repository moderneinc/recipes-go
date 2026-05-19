/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditJsonRawMessage(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditJsonRawMessage{})
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

func TestAuditJsonRawMessageNoChangeMarshal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditJsonRawMessage{})
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
