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
			)

			func write(v any) error {
				return json.MarshalWrite(os.Stdout, v)
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
			)

			func read(v any) error {
				return json.UnmarshalRead(os.Stdin, &v)
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
			)

			func write(v any) error {
				return j.MarshalWrite(os.Stdout, v)
			}
		`),
	)
}

// A file whose only json use is a plain Marshal has no construct this recipe
// rewrites, so it is left unchanged.
func TestNoChangePlainMarshal(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func run(v any) ([]byte, error) {
				return json.Marshal(v)
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
			)

			func write(v any) error {
				return json.MarshalWrite(os.Stdout, v)
			}

			func dump(v any) ([]byte, error) {
				return json.Marshal(v)
			}
		`),
	)
}

func TestNoChangeStoredEncoderVar(t *testing.T) {
	spec().RewriteRun(t,
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

func TestNoChangeAlreadyReferencesJsonSubpath(t *testing.T) {
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
			)

			type Point struct {
				X int
				Y int
			}

			func write() error {
				return json.MarshalWrite(os.Stdout, Point{X: 1, Y: 2})
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
			)

			func writeJSON(v any) error {
				return json.MarshalWrite(os.Stdout, v)
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
			)

			func write(v any) error {
				return json.MarshalWrite(os.Stdout, /* payload */ v)
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
