/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveRedundantTypeConversionString(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantTypeConversion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(s string) string {
				return string(s)
			}
		`, `
			package main

			func f(s string) string {
				return s
			}
		`),
	)
}

func TestRemoveRedundantTypeConversionFieldAccess(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantTypeConversion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type w struct{ n int64 }

			func f(x w) int64 {
				return int64(x.n)
			}
		`, `
			package main

			type w struct{ n int64 }

			func f(x w) int64 {
				return x.n
			}
		`),
	)
}

func TestRemoveRedundantTypeConversionUnsigned(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantTypeConversion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(u uint32) uint32 {
				return uint32(u)
			}
		`, `
			package main

			func f(u uint32) uint32 {
				return u
			}
		`),
	)
}

func TestRemoveRedundantTypeConversionKeepsRealConversions(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantTypeConversion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type celsius float32

			func f(b []byte, i int, c celsius) (string, int64, float32) {
				return string(b), int64(i), float32(c)
			}
		`),
	)
}

func TestRemoveRedundantTypeConversionIntAndFloat(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantTypeConversion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(i int, d float64) (int, float64) {
				return int(i), float64(d)
			}
		`, `
			package main

			func f(i int, d float64) (int, float64) {
				return i, d
			}
		`),
	)
}

// byte and uint8 name one type, as do rune and int32.
func TestRemoveRedundantTypeConversionAliasPair(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantTypeConversion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(u uint8, i int32) (byte, rune) {
				return byte(u), rune(i)
			}
		`, `
			package main

			func f(u uint8, i int32) (byte, rune) {
				return u, i
			}
		`),
	)
}

// Dropping the conversion around an untyped constant would change its type.
func TestRemoveRedundantTypeConversionKeepsLiteralConversion(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantTypeConversion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() string {
				return string("x")
			}
		`),
	)
}
