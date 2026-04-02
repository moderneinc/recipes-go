/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferErrorsIsNotExist(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferErrorsIsForOsCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(err error) bool {
				return os.IsNotExist(err)
			}
		`, `
			package main

			import "os"

			func f(err error) bool {
				return errors.Is(err, fs.ErrNotExist)
			}
		`),
	)
}

func TestPreferErrorsIsExist(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferErrorsIsForOsCheck{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(err error) bool {
				return os.IsExist(err)
			}
		`, `
			package main

			import "os"

			func f(err error) bool {
				return errors.Is(err, fs.ErrExist)
			}
		`),
	)
}
