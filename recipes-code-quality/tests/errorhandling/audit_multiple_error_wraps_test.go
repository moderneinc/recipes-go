/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditMultipleErrorWrapsReplaced(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AuditMultipleErrorWraps{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(err1, err2 error) error {
				return fmt.Errorf("a: %w, b: %w", err1, err2)
			}
		`, `
			package main

			import "fmt"

			func f(err1, err2 error) error {
				return fmt.Errorf("a: %w, b: %v", err1, err2)
			}
		`),
	)
}

func TestAuditMultipleErrorWrapsThreeW(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AuditMultipleErrorWraps{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(a, b, c error) error {
				return fmt.Errorf("x: %w, y: %w, z: %w", a, b, c)
			}
		`, `
			package main

			import "fmt"

			func f(a, b, c error) error {
				return fmt.Errorf("x: %w, y: %v, z: %v", a, b, c)
			}
		`),
	)
}

func TestAuditMultipleErrorWrapsNoChangeSingleW(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AuditMultipleErrorWraps{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(err error) error {
				return fmt.Errorf("failed: %w", err)
			}
		`),
	)
}

func TestAuditMultipleErrorWrapsNoChangeNoW(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.AuditMultipleErrorWraps{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(msg string) error {
				return fmt.Errorf("failed: %s", msg)
			}
		`),
	)
}
