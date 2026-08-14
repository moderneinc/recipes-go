/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/jsonv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func rawMessageSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&jsonv2.RelocateRawMessage{})
}

// When RawMessage is the only json use, it becomes jsontext.Value and the now
// unused encoding/json import is dropped rather than swapped to v2.
func TestRelocateRawMessageOnlyField(t *testing.T) {
	rawMessageSpec().RewriteRun(t,
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

// A json.RawMessage(x) conversion is rewritten to jsontext.Value(x).
func TestRelocateRawMessageConversion(t *testing.T) {
	rawMessageSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func wrap(b []byte) json.RawMessage {
				return json.RawMessage(b)
			}
		`, `
			package main

			import "encoding/json/jsontext"

			func wrap(b []byte) jsontext.Value {
				return jsontext.Value(b)
			}
		`),
	)
}

// A surviving Marshal keeps encoding/json referenced, so the import is renamed
// to v2 while RawMessage relocates to jsontext.
func TestRelocateRawMessageWithCoexistingPlainMarshal(t *testing.T) {
	rawMessageSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			type Payload struct {
				Raw json.RawMessage
			}

			func dump(p Payload) ([]byte, error) {
				return json.Marshal(p)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"encoding/json/jsontext"
			)

			type Payload struct {
				Raw jsontext.Value
			}

			func dump(p Payload) ([]byte, error) {
				return json.Marshal(p)
			}
		`),
	)
}

// A streaming chain is a removed symbol this recipe cannot rewrite, so a file
// mixing it with RawMessage is left unchanged.
func TestRelocateRawMessageNoChangeWithStreaming(t *testing.T) {
	rawMessageSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			type Payload struct {
				Raw json.RawMessage
			}

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}

// A file with no RawMessage has nothing for this recipe to rewrite.
func TestRelocateRawMessageNoChangeWithoutRawMessage(t *testing.T) {
	rawMessageSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func run(v any) ([]byte, error) {
				return json.Marshal(v)
			}
		`),
	)
}

// A dot import is left untouched.
func TestRelocateRawMessageNoChangeDotImport(t *testing.T) {
	rawMessageSpec().RewriteRun(t,
		test.Golang(`
			package main

			import . "encoding/json"

			type Payload struct {
				Raw RawMessage
			}
		`),
	)
}
