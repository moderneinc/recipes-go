/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindYamlUnmarshal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindYamlUnmarshal{})
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

func TestFindYamlUnmarshalNoChangeJsonUnmarshal(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindYamlUnmarshal{})
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
