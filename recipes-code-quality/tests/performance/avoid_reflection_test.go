/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidReflectionTypeOf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AvoidReflection{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "reflect"

			func f(x interface{}) {
				_ = reflect.TypeOf(x)
			}
		`, `
			package main

			import "reflect"

			func f(x interface{}) {
				_ =/*~~(reflection is slow; avoid in performance-sensitive code)~~>*/ reflect.TypeOf(x)
			}
		`),
	)
}

func TestAvoidReflectionValueOf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AvoidReflection{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "reflect"

			func f(x interface{}) {
				_ = reflect.ValueOf(x)
			}
		`, `
			package main

			import "reflect"

			func f(x interface{}) {
				_ =/*~~(reflection is slow; avoid in performance-sensitive code)~~>*/ reflect.ValueOf(x)
			}
		`),
	)
}

func TestAvoidReflectionNoChangeFmtCall(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.AvoidReflection{})
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
