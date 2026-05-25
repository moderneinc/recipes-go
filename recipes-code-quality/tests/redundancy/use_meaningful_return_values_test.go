/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseMeaningfulReturnValues(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.UseMeaningfulReturnValues{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() (error, error) {
				return nil, nil
			}
		`, `
			package main

			func f() (error, error) {
				/*~~(all return values are nil; possible missing error or result)~~>*/return nil, nil
			}
		`),
	)
}

func TestUseMeaningfulReturnValuesNoChangeNilAndErr(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.UseMeaningfulReturnValues{})
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

func TestUseMeaningfulReturnValuesNoChangeResultAndNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.UseMeaningfulReturnValues{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() (int, error) {
				return 0, nil
			}
		`),
	)
}

func TestUseMeaningfulReturnValuesNoChangeSingleNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.UseMeaningfulReturnValues{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() error {
				return nil
			}
		`),
	)
}
