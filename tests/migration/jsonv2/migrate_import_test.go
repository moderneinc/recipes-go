/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/jsonv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func migrateImportSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&jsonv2.MigrateImportToJSONV2{})
}

// The orchestrator applies streaming, MarshalIndent, and Encoder/Decoder
// rewrites together in one pass.
func TestMigrateImportMixedConstructs(t *testing.T) {
	migrateImportSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func stream(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}

			func indent(v any) ([]byte, error) {
				return json.MarshalIndent(v, "", "  ")
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"os"
				"encoding/json/jsontext"
			)

			func stream(v any) error {
				return json.MarshalWrite(os.Stdout, v)
			}

			func indent(v any) ([]byte, error) {
				return json.Marshal(v, jsontext.WithIndent("  "), jsontext.WithIndentPrefix(""))
			}
		`),
	)
}

// A stored Encoder/Decoder is not a mechanical construct this orchestrator
// migrates, so the file is left unchanged.
func TestMigrateImportNoChangeStoredEncoder(t *testing.T) {
	migrateImportSpec().RewriteRun(t,
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
		`),
	)
}

func TestMigrateImportNoChangePlainMarshal(t *testing.T) {
	migrateImportSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func run(v any) ([]byte, error) {
				return json.Marshal(v)
			}
		`),
	)
}

func TestMigrateImportNoChangeRawMessageField(t *testing.T) {
	migrateImportSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			type T struct {
				Raw json.RawMessage
			}

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}

func TestMigrateImportNoChangeCustomMarshalJSON(t *testing.T) {
	migrateImportSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			type T struct{}

			func (T) MarshalJSON() ([]byte, error) {
				return []byte("null"), nil
			}

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}
