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

// A removed package function (json.Compact) has no v2 rewrite, so it blocks the
// import swap and the whole file is left unchanged despite the streaming chain
// alongside it that would otherwise migrate.
func TestMigrateToJSONV2NoChangeWithRemovedFunc(t *testing.T) {
	bundleSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"bytes"
				"encoding/json"
				"os"
			)

			func run(v any, dst *bytes.Buffer, src []byte) error {
				if err := json.Compact(dst, src); err != nil {
					return err
				}
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}

// A removed type reference (json.Number, json.Delim, or a json error type) cannot
// survive the import swap, so the whole file is left unchanged despite the
// streaming chain that would otherwise migrate.
func TestMigrateToJSONV2NoChangeWithRemovedType(t *testing.T) {
	bundleSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			type T struct {
				N   json.Number
				D   json.Delim
				Err *json.SyntaxError
			}

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}

// A time.Duration field marshals to a runtime error under bare v2, so the
// default bundle leaves the file for review; only the compat bundle migrates it.
func TestMigrateToJSONV2NoChangeWithDurationField(t *testing.T) {
	bundleSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
				"time"
			)

			type T struct {
				Timeout time.Duration
			}

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}

// A file mixing two mechanical constructs is left unchanged, since neither the
// streaming nor the MarshalIndent recipe can swap the single json import without
// stranding the other's not-yet-rewritten construct.
func TestMigrateToJSONV2NoChangeMixedMechanical(t *testing.T) {
	bundleSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func dump(v any) ([]byte, error) {
				return json.MarshalIndent(v, "", "  ")
			}

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}
