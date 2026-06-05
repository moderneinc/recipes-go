/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindDeprecatedAtomicFunctionsAddInt32(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDeprecatedAtomicFunctions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sync/atomic"

			func f() {
				var x int32
				atomic.AddInt32(&x, 1)
			}
		`, `
			package main

			import "sync/atomic"

			func f() {
				var x int32/*~~(deprecated sync/atomic function; prefer the type-safe atomic types introduced in Go 1.19 (e.g. atomic.Int32))~~>*/
				atomic.AddInt32(&x, 1)
			}
		`),
	)
}

func TestFindDeprecatedAtomicFunctionsLoadInt64(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDeprecatedAtomicFunctions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sync/atomic"

			func f() {
				var x int64
				_ = atomic.LoadInt64(&x)
			}
		`, `
			package main

			import "sync/atomic"

			func f() {
				var x int64
				_ =/*~~(deprecated sync/atomic function; prefer the type-safe atomic types introduced in Go 1.19 (e.g. atomic.Int32))~~>*/ atomic.LoadInt64(&x)
			}
		`),
	)
}

func TestFindDeprecatedAtomicFunctionsNoChangeTypeSafe(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDeprecatedAtomicFunctions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sync/atomic"

			func f() {
				var x atomic.Int32
				x.Add(1)
			}
		`),
	)
}

func TestFindDeprecatedAtomicFunctionsNoChangeOtherPkg(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDeprecatedAtomicFunctions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() {
				fmt.Println("hello")
			}
		`),
	)
}

func TestFindDeprecatedAtomicFunctionsStoreInt32(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDeprecatedAtomicFunctions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sync/atomic"

			func f() {
				var x int32
				atomic.StoreInt32(&x, 42)
			}
		`, `
			package main

			import "sync/atomic"

			func f() {
				var x int32/*~~(deprecated sync/atomic function; prefer the type-safe atomic types introduced in Go 1.19 (e.g. atomic.Int32))~~>*/
				atomic.StoreInt32(&x, 42)
			}
		`),
	)
}

func TestFindDeprecatedAtomicFunctionsCompareAndSwapInt64(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindDeprecatedAtomicFunctions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "sync/atomic"

			func f() {
				var x int64
				atomic.CompareAndSwapInt64(&x, 0, 1)
			}
		`, `
			package main

			import "sync/atomic"

			func f() {
				var x int64/*~~(deprecated sync/atomic function; prefer the type-safe atomic types introduced in Go 1.19 (e.g. atomic.Int32))~~>*/
				atomic.CompareAndSwapInt64(&x, 0, 1)
			}
		`),
	)
}
