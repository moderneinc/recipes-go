/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/jsonv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func spec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&jsonv2.UseMarshalWriteUnmarshalRead{})
}

func TestMigrateEncoderStreaming(t *testing.T) {
	spec().RewriteRun(t,
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

func TestMigrateDecoderStreaming(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func read(v any) error {
				return json.NewDecoder(os.Stdin).Decode(&v)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"os"
				"encoding/json/jsontext"
			)

			func read(v any) error {
				return json.UnmarshalDecode(jsontext.NewDecoder(os.Stdin), &v)
			}
		`),
	)
}

func TestMigrateAliasedImport(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				j "encoding/json"
				"os"
			)

			func write(v any) error {
				return j.NewEncoder(os.Stdout).Encode(v)
			}
		`, `
			package main

			import (
				j "encoding/json/v2"
				"os"
				"encoding/json/jsontext"
			)

			func write(v any) error {
				return j.MarshalEncode(jsontext.NewEncoder(os.Stdout), v)
			}
		`),
	)
}

// Marshal survives in v2, so a plain Marshal sharing the file with a streaming
// chain stays put and adopts v2 semantics while the chain migrates.
func TestMigrateLeavesCoexistingPlainMarshal(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}

			func dump(v any) ([]byte, error) {
				return json.Marshal(v)
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

			func dump(v any) ([]byte, error) {
				return json.Marshal(v)
			}
		`),
	)
}

func TestNoChangeDotImport(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				. "encoding/json"
				"os"
			)

			func write(v any) error {
				return NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}

// The import swap rewrites only the exact encoding/json import, so a file that
// already imports encoding/json/jsontext migrates without corrupting it.
func TestMigrateAlongsideExistingJsontextImport(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"encoding/json/jsontext"
				"os"
			)

			var _ = jsontext.NewEncoder

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"encoding/json/jsontext"
				"os"
			)

			var _ = jsontext.NewEncoder

			func write(v any) error {
				return json.MarshalEncode(jsontext.NewEncoder(os.Stdout), v)
			}
		`),
	)
}

// A time.Duration field marshals to a runtime error under v2 without an explicit
// format, so the file is left for review rather than migrated.
func TestNoChangeWithDurationField(t *testing.T) {
	spec().RewriteRun(t,
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

// A file that already imports encoding/json/v2 is left unchanged, since swapping
// would produce a duplicate import.
func TestNoChangeWhenJsonV2AlreadyImported(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				jsonv2 "encoding/json/v2"
				"os"
			)

			var _ = jsonv2.Marshal

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}

// The value argument being a composite literal exercises the generic
// argument-spacing path (a space must follow the comma).
func TestMigrateCompositeLiteralValue(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			type Point struct {
				X int
				Y int
			}

			func write() error {
				return json.NewEncoder(os.Stdout).Encode(Point{X: 1, Y: 2})
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"os"
				"encoding/json/jsontext"
			)

			type Point struct {
				X int
				Y int
			}

			func write() error {
				return json.MarshalEncode(jsontext.NewEncoder(os.Stdout), Point{X: 1, Y: 2})
			}
		`),
	)
}

// The json chain migrates while an unrelated non-json encoder chain in the same
// file is left untouched.
func TestMigrateJsonLeavesXmlChain(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"encoding/xml"
				"os"
			)

			func writeJSON(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}

			func writeXML(v any) error {
				return xml.NewEncoder(os.Stdout).Encode(v)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"encoding/xml"
				"os"
				"encoding/json/jsontext"
			)

			func writeJSON(v any) error {
				return json.MarshalEncode(jsontext.NewEncoder(os.Stdout), v)
			}

			func writeXML(v any) error {
				return xml.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}

// A comment on the value argument is preserved, not dropped, when the call is
// rebuilt.
func TestMigratePreservesValueComment(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode( /* payload */ v)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"os"
				"encoding/json/jsontext"
			)

			func write(v any) error {
				return json.MarshalEncode(jsontext.NewEncoder(os.Stdout), /* payload */ v)
			}
		`),
	)
}

func TestNoChangeBlankImport(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				_ "encoding/json"
				"os"
			)

			func main() {
				_ = os.Stdout
			}
		`),
	)
}
