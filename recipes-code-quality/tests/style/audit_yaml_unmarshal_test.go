/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditYamlUnmarshal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditYamlUnmarshal{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "gopkg.in/yaml.v3"

			func f() {
				data := []byte("key: value")
				var out map[string]string
				_ = yaml.Unmarshal(data, &out)
			}
		`),
	)
}

func TestAuditYamlUnmarshalNoChangeJsonUnmarshal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditYamlUnmarshal{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func f() {
				data := []byte("{}")
				var out map[string]string
				_ = json.Unmarshal(data, &out)
			}
		`),
	)
}
