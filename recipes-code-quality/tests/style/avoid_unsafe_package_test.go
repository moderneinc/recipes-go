/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidUnsafePackage(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidUnsafePackage{})
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

func TestAvoidUnsafePackageNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidUnsafePackage{})
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
