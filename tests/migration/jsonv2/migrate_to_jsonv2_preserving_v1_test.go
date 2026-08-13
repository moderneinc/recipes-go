/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/jsonv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func preservingBundleSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&jsonv2.MigrateToJSONV2PreservingV1{})
}

// The one-step compatibility bundle runs the mechanical migration and then
// appends jsonv1.DefaultOptionsV1() so the migrated output stays byte-identical
// to v1.
func TestMigrateToJSONV2PreservingV1(t *testing.T) {
	preservingBundleSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"os"
				"encoding/json/jsontext"
				jsonv1 "encoding/json"
			)

			func write(v any) error {
				return json.MarshalEncode(jsontext.NewEncoder(os.Stdout, jsonv1.DefaultOptionsV1()), v)
			}
		`),
	)
}
