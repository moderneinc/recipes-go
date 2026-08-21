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

// int and float64 share their attributed type with int32 and complex128, so a
// conversion to either is left alone even when it looks redundant.
func TestRemoveRedundantTypeConversionSkipsAmbiguousTargets(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantTypeConversion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(i int, d float64) (int, float64) {
				return int(i), float64(d)
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

func TestRemoveRedundantTypeConversionKeepsQualifiedUntypedConstConversion(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantTypeConversion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "math"

			func sink(v any) {}

			func f() {
				sink(uint16(math.MaxUint16))
			}
		`),
	)
}

func TestRemoveRedundantTypeConversionKeepsPackageUntypedConstConversion(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantTypeConversion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const maxU16 = 1<<16 - 1

			func sink(v any) {}

			func f() {
				sink(uint16(maxU16))
			}
		`),
	)
}

func TestRemoveRedundantTypeConversionKeepsNumericLiteralConversion(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantTypeConversion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func sink(v any) {}

			func f() {
				sink(uint16(65535))
			}
		`),
	)
}

func TestRemoveRedundantTypeConversionTypedConst(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveRedundantTypeConversion{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			const maxU16 uint16 = 1<<16 - 1

			func f() uint16 {
				return uint16(maxU16)
			}
		`, `
			package main

			const maxU16 uint16 = 1<<16 - 1

			func f() uint16 {
				return maxU16
			}
		`),
	)
}
