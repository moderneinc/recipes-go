/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindAllNilReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindAllNilReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() (error, error) {
				return nil, nil
			}
		`),
	)
}

func TestFindAllNilReturnNoChangeNilAndErr(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindAllNilReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "errors"

			func f() (interface{}, error) {
				err := errors.New("fail")
				return nil, err
			}
		`),
	)
}

func TestFindAllNilReturnNoChangeResultAndNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindAllNilReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() (int, error) {
				return 0, nil
			}
		`),
	)
}

func TestFindAllNilReturnNoChangeSingleNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindAllNilReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() error {
				return nil
			}
		`),
	)
}
