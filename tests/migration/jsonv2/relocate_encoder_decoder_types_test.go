/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/jsonv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func relocateSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&jsonv2.RelocateEncoderDecoderTypes{})
}

// The escape analysis is scoped per function, so a codec variable name reused
// across functions is migrated in each.
func TestRelocateReusedNameAcrossFunctions(t *testing.T) {
	relocateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func a(v any) error {
				enc := json.NewEncoder(os.Stdout)
				return enc.Encode(v)
			}

			func b(v any) error {
				enc := json.NewEncoder(os.Stderr)
				return enc.Encode(v)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"os"
				"encoding/json/jsontext"
			)

			func a(v any) error {
				enc := jsontext.NewEncoder(os.Stdout)
				return json.MarshalEncode(enc, v)
			}

			func b(v any) error {
				enc := jsontext.NewEncoder(os.Stderr)
				return json.MarshalEncode(enc, v)
			}
		`),
	)
}

// One function's encoder escapes, so the whole file is left unchanged even
// though another function's same-named encoder is clean.
func TestRelocateNoChangeOneFunctionEscapes(t *testing.T) {
	relocateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func a(v any) error {
				enc := json.NewEncoder(os.Stdout)
				return enc.Encode(v)
			}

			func b(v any) error {
				enc := json.NewEncoder(os.Stderr)
				sink(enc)
				return enc.Encode(v)
			}

			func sink(x any) {}
		`),
	)
}

// A local encoder used across multiple Encode calls is the case the streaming
// recipe cannot handle.
func TestRelocateStoredEncoderMultipleCalls(t *testing.T) {
	relocateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func write(a, b any) error {
				enc := json.NewEncoder(os.Stdout)
				if err := enc.Encode(a); err != nil {
					return err
				}
				return enc.Encode(b)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"os"
				"encoding/json/jsontext"
			)

			func write(a, b any) error {
				enc := jsontext.NewEncoder(os.Stdout)
				if err := json.MarshalEncode(enc, a); err != nil {
					return err
				}
				return json.MarshalEncode(enc, b)
			}
		`),
	)
}

func TestRelocateStoredDecoder(t *testing.T) {
	relocateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func read(v any) error {
				dec := json.NewDecoder(os.Stdin)
				return dec.Decode(&v)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"os"
				"encoding/json/jsontext"
			)

			func read(v any) error {
				dec := jsontext.NewDecoder(os.Stdin)
				return json.UnmarshalDecode(dec, &v)
			}
		`),
	)
}

// The encoder escapes by being passed to another function, so the value is left
// unchanged.
func TestRelocateNoChangeEncoderEscapesToCall(t *testing.T) {
	relocateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func write(v any) error {
				enc := json.NewEncoder(os.Stdout)
				sink(enc)
				return enc.Encode(v)
			}

			func sink(x any) {}
		`),
	)
}

// The encoder is taken as a method value, so the value is left unchanged.
func TestRelocateNoChangeMethodValue(t *testing.T) {
	relocateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func write(v any) error {
				enc := json.NewEncoder(os.Stdout)
				_ = enc.Encode
				return enc.Encode(v)
			}
		`),
	)
}

func TestRelocateNoChangeEmbeddedEncoder(t *testing.T) {
	relocateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			type MyEnc struct {
				*json.Encoder
			}

			func write(v any) error {
				m := &MyEnc{json.NewEncoder(os.Stdout)}
				return m.Encode(v)
			}
		`),
	)
}

func TestRelocateNoChangeTypedParameter(t *testing.T) {
	relocateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func write(enc *json.Encoder, v any) error {
				return enc.Encode(v)
			}
		`),
	)
}

// Marshal survives in v2, so a plain Marshal sharing the file with a local
// encoder stays put and adopts v2 semantics while the encoder relocates.
func TestRelocateLeavesCoexistingPlainMarshal(t *testing.T) {
	relocateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func run(v any) error {
				enc := json.NewEncoder(os.Stdout)
				if _, err := json.Marshal(v); err != nil {
					return err
				}
				return enc.Encode(v)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"os"
				"encoding/json/jsontext"
			)

			func run(v any) error {
				enc := jsontext.NewEncoder(os.Stdout)
				if _, err := json.Marshal(v); err != nil {
					return err
				}
				return json.MarshalEncode(enc, v)
			}
		`),
	)
}

// A fluent chain has no local encoder to relocate; the streaming recipe handles
// it instead.
func TestRelocateNoChangeFluentChain(t *testing.T) {
	relocateSpec().RewriteRun(t,
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

// A configured encoder uses a method this recipe does not rewrite, so the file
// is left unchanged.
func TestRelocateNoChangeSetIndent(t *testing.T) {
	relocateSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func write(v any) error {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(v)
			}
		`),
	)
}
