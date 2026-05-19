/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUsePackageLevelErrorSentinel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UsePackageLevelErrorSentinel{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			func f() error {
				return errors.New("not found")
			}
		`, `
			package main

			import "errors"

			var ErrNotFound = errors.New("not found")

			func f() error {
				return ErrNotFound
			}
		`),
	)
}

func TestUsePackageLevelErrorSentinelNoChangeFmtErrorf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UsePackageLevelErrorSentinel{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() error {
				return fmt.Errorf("fail")
			}
		`),
	)
}

func TestUsePackageLevelErrorSentinelAlreadyAtPackageLevel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UsePackageLevelErrorSentinel{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			var ErrNotFound = errors.New("not found")

			func f() error {
				return ErrNotFound
			}
		`),
	)
}
