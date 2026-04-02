/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseErrorsNewSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseErrorsNewForSimpleErrors{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() error {
				return fmt.Errorf("something went wrong")
			}
		`, `
			package main

			import "fmt"

			func f() error {
				return errors.New("something went wrong")
			}
		`),
	)
}

func TestUseErrorsNewNoChangeFormatVerbS(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseErrorsNewForSimpleErrors{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(msg string) error {
				return fmt.Errorf("error: %s", msg)
			}
		`),
	)
}

func TestUseErrorsNewNoChangeFormatVerbD(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseErrorsNewForSimpleErrors{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(n int) error {
				return fmt.Errorf("count: %d", n)
			}
		`),
	)
}
