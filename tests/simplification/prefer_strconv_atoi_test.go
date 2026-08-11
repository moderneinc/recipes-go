/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferStrconvAtoi(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStrconvAtoi{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strconv"

			func f(s string) int {
				n, _ := strconv.ParseInt(s, 10, 0)
				return int(n)
			}
		`, `
			package main

			import "strconv"

			func f(s string) int {
				n, _ := strconv.Atoi(s)
				return int(n)
			}
		`),
	)
}

// Skips the capture-then-return pattern where n is returned as int64.
func TestPreferStrconvAtoiNoChangeCapturedReturnedInt64(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStrconvAtoi{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strconv"

			func parse(s string) (int64, error) {
				n, err := strconv.ParseInt(s, 10, 0)
				if err != nil {
					return 0, err
				}
				return n, nil
			}
		`),
	)
}

// Rewrites when the captured value is only used through an int() conversion.
func TestPreferStrconvAtoiCapturedConvertedToInt(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStrconvAtoi{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strconv"

			func parse(s string) (int, error) {
				n, err := strconv.ParseInt(s, 10, 0)
				if err != nil {
					return 0, err
				}
				return int(n), nil
			}
		`, `
			package main

			import "strconv"

			func parse(s string) (int, error) {
				n, err := strconv.Atoi(s)
				if err != nil {
					return 0, err
				}
				return int(n), nil
			}
		`),
	)
}

func TestPreferStrconvAtoiNoChangeBase16(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStrconvAtoi{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strconv"

			func f(s string) (int64, error) {
				return strconv.ParseInt(s, 16, 0)
			}
		`),
	)
}

func TestPreferStrconvAtoiNoChangeBitSize(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStrconvAtoi{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strconv"

			func f(s string) (int64, error) {
				return strconv.ParseInt(s, 10, 64)
			}
		`),
	)
}

// Skips a direct return of the int64 result.
func TestPreferStrconvAtoiNoChangeInt64Context(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStrconvAtoi{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "strconv"

			func f(s string) (int64, error) {
				return strconv.ParseInt(s, 10, 0)
			}
		`),
	)
}
