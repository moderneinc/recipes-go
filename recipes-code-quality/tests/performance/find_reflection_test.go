/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindReflectionTypeOf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindReflection{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "reflect"

			func f(x interface{}) {
				_ = reflect.TypeOf(x)
			}
		`),
	)
}

func TestFindReflectionValueOf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindReflection{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "reflect"

			func f(x interface{}) {
				_ = reflect.ValueOf(x)
			}
		`),
	)
}

func TestFindReflectionNoChangeFmtCall(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.FindReflection{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(x interface{}) {
				fmt.Println(x)
			}
		`),
	)
}
