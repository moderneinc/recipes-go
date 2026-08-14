/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/jsonv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func importOnlySpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&jsonv2.MigrateImportOnlyToJSONV2{})
}

// A plain Marshal exists unchanged in v2, so only the import is swapped and the
// call adopts v2 semantics.
func TestMigrateImportOnlyPlainMarshal(t *testing.T) {
	importOnlySpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func run(v any) ([]byte, error) {
				return json.Marshal(v)
			}
		`, `
			package main

			import "encoding/json/v2"

			func run(v any) ([]byte, error) {
				return json.Marshal(v)
			}
		`),
	)
}

// A surviving Marshaler type reference is enough to migrate the import.
func TestMigrateImportOnlyMarshalerParam(t *testing.T) {
	importOnlySpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func encode(m json.Marshaler) ([]byte, error) {
				return m.MarshalJSON()
			}
		`, `
			package main

			import "encoding/json/v2"

			func encode(m json.Marshaler) ([]byte, error) {
				return m.MarshalJSON()
			}
		`),
	)
}

func TestMigrateImportOnlyAliasedImport(t *testing.T) {
	importOnlySpec().RewriteRun(t,
		test.Golang(`
			package main

			import j "encoding/json"

			func run(v any) ([]byte, error) {
				return j.Marshal(v)
			}
		`, `
			package main

			import j "encoding/json/v2"

			func run(v any) ([]byte, error) {
				return j.Marshal(v)
			}
		`),
	)
}

// A streaming call is a removed symbol, so the file is left for the streaming
// recipe rather than swapped here.
func TestMigrateImportOnlyNoChangeWithStreaming(t *testing.T) {
	importOnlySpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}

// A RawMessage field is a removed type, so the file is left for RelocateRawMessage.
func TestMigrateImportOnlyNoChangeWithRawMessage(t *testing.T) {
	importOnlySpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			type T struct {
				Raw json.RawMessage
			}

			func run(v any) ([]byte, error) {
				return json.Marshal(v)
			}
		`),
	)
}

// A dot import is left untouched.
func TestMigrateImportOnlyNoChangeDotImport(t *testing.T) {
	importOnlySpec().RewriteRun(t,
		test.Golang(`
			package main

			import . "encoding/json"

			func run(v any) ([]byte, error) {
				return Marshal(v)
			}
		`),
	)
}
