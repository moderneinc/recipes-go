/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/jsonv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func bundleSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&jsonv2.MigrateToJSONV2{})
}

func TestMigrateToJSONV2Streaming(t *testing.T) {
	bundleSpec().RewriteRun(t,
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
			)

			func write(v any) error {
				return json.MarshalEncode(jsontext.NewEncoder(os.Stdout), v)
			}
		`),
	)
}

func TestMigrateToJSONV2MarshalIndent(t *testing.T) {
	bundleSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func dump(data any) ([]byte, error) {
				return json.MarshalIndent(data, "", "  ")
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"encoding/json/jsontext"
			)

			func dump(data any) ([]byte, error) {
				return json.Marshal(data, jsontext.WithIndent("  "), jsontext.WithIndentPrefix(""))
			}
		`),
	)
}

// A stored encoder is handled only by the RelocateEncoderDecoderTypes member of
// the bundle, so this confirms the composition covers it.
func TestMigrateToJSONV2StoredEncoder(t *testing.T) {
	bundleSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func write(v any) error {
				enc := json.NewEncoder(os.Stdout)
				return enc.Encode(v)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"os"
				"encoding/json/jsontext"
			)

			func write(v any) error {
				enc := jsontext.NewEncoder(os.Stdout)
				return json.MarshalEncode(enc, v)
			}
		`),
	)
}

// A json.RawMessage field is handled only by the RelocateRawMessage member of
// the bundle, so this confirms the composition covers it.
func TestMigrateToJSONV2RawMessage(t *testing.T) {
	bundleSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			type Payload struct {
				Raw json.RawMessage
			}
		`, `
			package main

			import "encoding/json/jsontext"

			type Payload struct {
				Raw jsontext.Value
			}
		`),
	)
}

// A plain Marshal file is handled only by the MigrateImportOnlyToJSONV2 member
// of the bundle, which swaps the import so the call adopts v2 semantics.
func TestMigrateToJSONV2PlainMarshal(t *testing.T) {
	bundleSpec().RewriteRun(t,
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
