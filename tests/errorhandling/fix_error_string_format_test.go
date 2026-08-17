/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFixErrorStringFormatCapitalized(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FixErrorStringFormat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			func f() error {
				return errors.New("Failed to connect")
			}
		`, `
			package main

			import "errors"

			func f() error {
				return errors.New("failed to connect")
			}
		`),
	)
}

func TestFixErrorStringFormatTrailingPunctuation(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FixErrorStringFormat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() error {
				return fmt.Errorf("connection failed.")
			}
		`, `
			package main

			import "fmt"

			func f() error {
				return fmt.Errorf("connection failed")
			}
		`),
	)
}

func TestFixErrorStringFormatBothIssues(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FixErrorStringFormat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			func f() error {
				return errors.New("Connection failed:")
			}
		`, `
			package main

			import "errors"

			func f() error {
				return errors.New("connection failed")
			}
		`),
	)
}

func TestFixErrorStringFormatKeepsWrappingVerb(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FixErrorStringFormat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(err error) error {
				return fmt.Errorf("Reading config: %w", err)
			}
		`, `
			package main

			import "fmt"

			func f(err error) error {
				return fmt.Errorf("reading config: %w", err)
			}
		`),
	)
}

// An initialism is not an ordinary capitalized word, and an ellipsis is
// deliberate, so neither is rewritten.
func TestFixErrorStringFormatNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.FixErrorStringFormat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"errors"
				"fmt"
			)

			func f() error {
				if false {
					return errors.New("HTTP request failed")
				}
				if false {
					return errors.New("waiting for lock...")
				}
				return fmt.Errorf("connection failed")
			}
		`),
	)
}
