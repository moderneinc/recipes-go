/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditErrorStringFormatCapitalized(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AuditErrorStringFormat{})
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
				return /*~~(error string should not be capitalized (ST1005))~~>*/errors.New("Failed to connect")
			}
		`),
	)
}

func TestAuditErrorStringFormatTrailingPunctuation(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AuditErrorStringFormat{})
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
				return /*~~(error string should not end with punctuation (ST1005))~~>*/fmt.Errorf("connection failed.")
			}
		`),
	)
}

func TestAuditErrorStringFormatCleanNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AuditErrorStringFormat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			func f() error {
				return errors.New("failed to connect")
			}
		`),
	)
}

func TestAuditErrorStringFormatInitialismNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AuditErrorStringFormat{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			func f() error {
				return errors.New("TLS handshake failed")
			}
		`),
	)
}
