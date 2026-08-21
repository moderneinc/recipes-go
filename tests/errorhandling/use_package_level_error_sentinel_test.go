/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/errorhandling"
	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
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

func TestUsePackageLevelErrorSentinelDeclaresSharedMessageOnce(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UsePackageLevelErrorSentinel{})
	spec.RewriteRun(t,
		test.Golang(`
			package sentinel

			import "errors"

			// A returns an error.
			func A() error {
				return errors.New("can't duplicate this flag")
			}
		`, `
			package sentinel

			import "errors"

			var ErrCantDuplicateThisFlag = errors.New("can't duplicate this flag")

			// A returns an error.
			func A() error {
				return ErrCantDuplicateThisFlag
			}
		`).WithPath("sentinel/a.go"),
		test.Golang(`
			package sentinel

			import "errors"

			// B returns an error.
			func B() error {
				return errors.New("can't duplicate this flag")
			}
		`, `
			package sentinel

			// B returns an error.
			func B() error {
				return ErrCantDuplicateThisFlag
			}
		`).WithPath("sentinel/b.go"),
	)
}

func TestUsePackageLevelErrorSentinelReusesExistingSentinel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UsePackageLevelErrorSentinel{})
	spec.RewriteRun(t,
		test.Golang(`
			package sentinel

			import "errors"

			var errNotFound = errors.New("not found")
		`).WithPath("sentinel/errors.go"),
		test.Golang(`
			package sentinel

			import "errors"

			func B() error {
				return errors.New("not found")
			}
		`, `
			package sentinel

			func B() error {
				return errNotFound
			}
		`).WithPath("sentinel/b.go"),
	)
}

func TestUsePackageLevelErrorSentinelNoChangeWhenNameIsTaken(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UsePackageLevelErrorSentinel{})
	spec.RewriteRun(t,
		test.Golang(`
			package sentinel

			import "errors"

			type ErrNotFound struct{}

			func B() error {
				return errors.New("not found")
			}
		`).WithPath("sentinel/b.go"),
	)
}

func TestUsePackageLevelErrorSentinelNoChangeAcrossDifferentBuildConstraints(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UsePackageLevelErrorSentinel{})
	spec.RewriteRun(t,
		test.Golang(`
			//go:build go1.16

			package sentinel

			import "errors"

			func A() error {
				return errors.New("not found")
			}
		`).WithPath("sentinel/a.go"),
		test.Golang(`
			//go:build go1.18

			package sentinel

			import "errors"

			func B() error {
				return errors.New("not found")
			}
		`).WithPath("sentinel/b.go"),
	)
}

func TestUsePackageLevelErrorSentinelDeclaresWithinOneBuildConstraint(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UsePackageLevelErrorSentinel{})
	spec.RewriteRun(t,
		test.Golang(`
			package sentinel

			import "errors"

			func A() error {
				return errors.New("not found")
			}
		`, `
			package sentinel

			import "errors"

			var ErrNotFound = errors.New("not found")

			func A() error {
				return ErrNotFound
			}
		`).WithPath("sentinel/a_test.go"),
		test.Golang(`
			package sentinel

			import "errors"

			func B() error {
				return errors.New("not found")
			}
		`, `
			package sentinel

			func B() error {
				return ErrNotFound
			}
		`).WithPath("sentinel/b_test.go"),
	)
}

func TestUsePackageLevelErrorSentinelDeclaresOutsideTheTestBuild(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.UsePackageLevelErrorSentinel{})
	spec.RewriteRun(t,
		test.Golang(`
			package sentinel

			import "errors"

			func A() error {
				return errors.New("not found")
			}
		`, `
			package sentinel

			func A() error {
				return ErrNotFound
			}
		`).WithPath("sentinel/a_test.go"),
		test.Golang(`
			package sentinel

			import "errors"

			func B() error {
				return errors.New("not found")
			}
		`, `
			package sentinel

			import "errors"

			var ErrNotFound = errors.New("not found")

			func B() error {
				return ErrNotFound
			}
		`).WithPath("sentinel/b.go"),
	)
}

// useErrorsNewThenSentinel is the catalog ordering that reaches a `fmt`-only
// file: `fmt.Errorf` with no verb becomes `errors.New`, which the sentinel
// recipe then hoists into a declaration the file has to import `errors` for.
type useErrorsNewThenSentinel struct{ recipe.Base }

func (r *useErrorsNewThenSentinel) Name() string        { return "test.UseErrorsNewThenSentinel" }
func (r *useErrorsNewThenSentinel) DisplayName() string { return "Use errors.New, then hoist" }
func (r *useErrorsNewThenSentinel) Description() string { return "Test-only recipe pairing." }
func (r *useErrorsNewThenSentinel) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		&style.UseErrorsNewForSimpleErrors{},
		&errorhandling.UsePackageLevelErrorSentinel{},
	}
}

func TestUsePackageLevelErrorSentinelImportsErrors(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&useErrorsNewThenSentinel{})
	spec.RewriteRun(t,
		test.Golang(`
			package sentinel

			import "fmt"

			func A() error {
				return fmt.Errorf("can't duplicate this flag")
			}

			func Shout(s string) string {
				return fmt.Sprintf("%s!", s)
			}
		`, `
			package sentinel

			import (
				"fmt"
				"errors"
			)

			var ErrCantDuplicateThisFlag = errors.New("can't duplicate this flag")

			func A() error {
				return ErrCantDuplicateThisFlag
			}

			func Shout(s string) string {
				return fmt.Sprintf("%s!", s)
			}
		`).WithPath("sentinel/a.go"),
	)
}
