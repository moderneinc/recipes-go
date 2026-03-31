/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindUnsafeUsage(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindUnsafeUsage{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "unsafe"

			func f() {
				var x int
				_ = unsafe.Pointer(&x)
			}
		`),
	)
}

func TestFindUnsafeUsageNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindUnsafeUsage{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() {
				fmt.Println("safe")
			}
		`),
	)
}
