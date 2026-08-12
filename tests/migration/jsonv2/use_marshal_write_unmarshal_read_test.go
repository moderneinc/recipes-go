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

func TestMigrateBothInOneFile(t *testing.T) {
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

			func read(v any) error {
				return json.NewDecoder(os.Stdin).Decode(&v)
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

// A local `[N]byte` or `time.Duration` is only a v2 concern as a struct field,
// so their presence outside a struct does not block migration.
func TestMigrateWithNonFieldByteArrayAndDuration(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
				"time"
			)

			func write(v any, timeout time.Duration) error {
				var scratch [8]byte
				_ = scratch
				_ = timeout
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"os"
				"time"
			)

			func write(v any, timeout time.Duration) error {
				var scratch [8]byte
				_ = scratch
				_ = timeout
				return json.MarshalWrite(os.Stdout, v)
			}
		`),
	)
}

// A migratable call sharing a file with json.MarshalIndent is left untouched:
// the whole file is atomic, so the import is never swapped out from under the
// still-v1 MarshalIndent call.
func TestNoChangeMixedWithMarshalIndent(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func run(v any) error {
				if err := json.NewEncoder(os.Stdout).Encode(v); err != nil {
					return err
				}
				_, err := json.MarshalIndent(v, "", "  ")
				return err
			}
		`),
	)
}

func TestNoChangeCustomMarshalJSON(t *testing.T) {
	spec().RewriteRun(t,
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

func TestNoChangeOmitemptyTag(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			type T struct {
				Name string `+"`"+`json:"name,omitempty"`+"`"+`
			}

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}

func TestNoChangeRawMessageField(t *testing.T) {
	spec().RewriteRun(t,
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

func TestNoChangeByteArrayField(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			type T struct {
				Hash [32]byte
			}

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}

func TestNoChangeDurationField(t *testing.T) {
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

func TestNoChangeByteArrayUint8Field(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			type T struct {
				Hash [32]uint8
			}

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}

func TestNoChangeAliasedDurationField(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
				tt "time"
			)

			type T struct {
				Timeout tt.Duration
			}

			func write(v any) error {
				return json.NewEncoder(os.Stdout).Encode(v)
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

func TestNoChangeNonJsonEncoder(t *testing.T) {
	spec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/xml"
				"os"
			)

			func write(v any) error {
				return xml.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}
