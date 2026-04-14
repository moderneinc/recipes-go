/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseAtomicTypesAddInt32(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseAtomicTypes{})
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
				var x int32
				x.Add(1)
			}
		`),
	)
}

func TestUseAtomicTypesLoadInt64(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseAtomicTypes{})
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
				_ = x.Load()
			}
		`),
	)
}

func TestUseAtomicTypesNoChangeTypeSafe(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseAtomicTypes{})
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

func TestUseAtomicTypesNoChangeOtherPkg(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseAtomicTypes{})
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

func TestUseAtomicTypesStoreInt32(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseAtomicTypes{})
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
				var x int32
				x.Store(42)
			}
		`),
	)
}

func TestUseAtomicTypesCompareAndSwapInt64(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseAtomicTypes{})
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
				var x int64
				x.CompareAndSwap(0, 1)
			}
		`),
	)
}
