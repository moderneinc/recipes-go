/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/jsonv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func preserveSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&jsonv2.PreserveV1Semantics{})
}

// A v2 marshal call gains jsonv1.DefaultOptionsV1() and the aliased encoding/json
// import so its output matches v1.
func TestPreserveV1SemanticsAppendsOption(t *testing.T) {
	preserveSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json/v2"

			func run(v any) ([]byte, error) {
				return json.Marshal(v)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				jsonv1 "encoding/json"
			)

			func run(v any) ([]byte, error) {
				return json.Marshal(v, jsonv1.DefaultOptionsV1())
			}
		`),
	)
}

// A call that already passes DefaultOptionsV1 is left unchanged.
func TestPreserveV1SemanticsIdempotent(t *testing.T) {
	preserveSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json/v2"
				jsonv1 "encoding/json"
			)

			func run(v any) ([]byte, error) {
				return json.Marshal(v, jsonv1.DefaultOptionsV1())
			}
		`),
	)
}

// A file still on encoding/json (v1) is left unchanged, since the recipe only
// runs on encoding/json/v2 files.
func TestPreserveV1SemanticsNoChangeOnV1File(t *testing.T) {
	preserveSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func run(v any) ([]byte, error) {
				return json.Marshal(v)
			}
		`),
	)
}
