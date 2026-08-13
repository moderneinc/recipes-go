/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/jsonv2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func marshalIndentSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&jsonv2.ReplaceMarshalIndent{})
}

func TestReplaceMarshalIndent(t *testing.T) {
	marshalIndentSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func dump(v any) ([]byte, error) {
				return json.MarshalIndent(v, "", "  ")
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"encoding/json/jsontext"
			)

			func dump(v any) ([]byte, error) {
				return json.Marshal(v, jsontext.WithIndent("  "), jsontext.WithIndentPrefix(""))
			}
		`),
	)
}

// The value argument being a pointer expression is carried through unchanged.
func TestReplaceMarshalIndentPointerValue(t *testing.T) {
	marshalIndentSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func dump(out any) ([]byte, error) {
				return json.MarshalIndent(&out, "", "  ")
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"encoding/json/jsontext"
			)

			func dump(out any) ([]byte, error) {
				return json.Marshal(&out, jsontext.WithIndent("  "), jsontext.WithIndentPrefix(""))
			}
		`),
	)
}

// Marshal survives in v2, so a plain Marshal sharing the file with a
// MarshalIndent stays put and adopts v2 semantics while MarshalIndent migrates.
func TestReplaceMarshalIndentLeavesCoexistingPlainMarshal(t *testing.T) {
	marshalIndentSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func run(v any) ([]byte, []byte, error) {
				a, err := json.Marshal(v)
				if err != nil {
					return nil, nil, err
				}
				b, err := json.MarshalIndent(v, "", "  ")
				return a, b, err
			}
		`, `
			package main

			import (
				"encoding/json/v2"
				"encoding/json/jsontext"
			)

			func run(v any) ([]byte, []byte, error) {
				a, err := json.Marshal(v)
				if err != nil {
					return nil, nil, err
				}
				b, err := json.Marshal(v, jsontext.WithIndent("  "), jsontext.WithIndentPrefix(""))
				return a, b, err
			}
		`),
	)
}

// The rewrite reorders the prefix and indent arguments, so a MarshalIndent
// carrying an argument comment is left unchanged rather than scrambling it.
func TestReplaceMarshalIndentNoChangeWithArgComment(t *testing.T) {
	marshalIndentSpec().RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func dump(v any) ([]byte, error) {
				return json.MarshalIndent(v, "" /* prefix */, "  ")
			}
		`),
	)
}

func TestReplaceMarshalIndentNoChangeWithStreaming(t *testing.T) {
	marshalIndentSpec().RewriteRun(t,
		test.Golang(`
			package main

			import (
				"encoding/json"
				"os"
			)

			func run(v any) error {
				if _, err := json.MarshalIndent(v, "", "  "); err != nil {
					return err
				}
				return json.NewEncoder(os.Stdout).Encode(v)
			}
		`),
	)
}
