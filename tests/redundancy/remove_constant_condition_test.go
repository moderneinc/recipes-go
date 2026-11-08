/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveConstantConditionTrue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveConstantCondition{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				if true {
					println("always")
				}
			}
		`, `
			package main

			func f() {
				{
					println("always")
				}
			}
		`),
	)
}

func TestRemoveConstantConditionFalse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveConstantCondition{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				if false {
					println("never")
				}
			}
		`, `
			package main

			func f() {
			}
		`),
	)
}

func TestRemoveConstantConditionFalseWithElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveConstantCondition{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() {
				if false {
					println("never")
				} else {
					println("always")
				}
			}
		`, `
			package main

			func f() {
				{
					println("always")
				}
			}
		`),
	)
}

func TestRemoveConstantConditionNoChangeVariable(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveConstantCondition{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ok bool) {
				if ok {
					println("maybe")
				}
			}
		`),
	)
}

// Synthetic semantic counterexample from the Sonar safety audit; not corpus evidence.
func TestRemoveConstantConditionPreservesShadowedTrue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveConstantCondition{})
	spec.RewriteRun(t, test.Golang(`package p
func f() { true := false; if true { println(1) } }`))
}

// Synthetic semantic counterexample from the Sonar safety audit; not corpus evidence.
func TestRemoveConstantConditionPreservesShadowedFalse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveConstantCondition{})
	spec.RewriteRun(t, test.Golang(`package p
func f(false bool) { if false { println(1) } }`))
}

// Synthetic semantic counterexample from the Sonar safety audit; not corpus evidence.
func TestRemoveConstantConditionPreservesInitializer(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveConstantCondition{})
	spec.RewriteRun(t, test.Golang(`package p
func f() { if x := 1; true { println(x) } }`))
}

// Synthetic semantic counterexample from the Sonar safety audit; not corpus evidence.
func TestRemoveConstantConditionPreservesReachableLabel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveConstantCondition{})
	spec.RewriteRun(t, test.Golang(`package p
func f(b bool) { if b { goto L }; if false { L: println(1) } }`))
}
