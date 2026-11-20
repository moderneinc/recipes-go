/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveDuplicateConditionsSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else if x > 0 {
					println("b")
				}
			}
		`, `
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				}
			}
		`),
	)
}

func TestRemoveDuplicateConditionsWithElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else if x > 0 {
					println("b")
				} else {
					println("c")
				}
			}
		`, `
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else {
					println("c")
				}
			}
		`),
	)
}

func TestRemoveDuplicateConditionsMiddleBranch(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else if x < 0 {
					println("b")
				} else if x > 0 {
					println("c")
				} else {
					println("d")
				}
			}
		`, `
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else if x < 0 {
					println("b")
				} else {
					println("d")
				}
			}
		`),
	)
}

func TestRemoveDuplicateConditionsNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else if x < 0 {
					println("b")
				} else {
					println("c")
				}
			}
		`),
	)
}

func TestRemoveDuplicateConditionsNoChangeNoElseIf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) {
				if x > 0 {
					println("a")
				} else {
					println("b")
				}
			}
		`),
	)
}

// Synthetic semantic counterexample from the Sonar safety audit; not corpus evidence.
func TestRemoveDuplicateConditionsPreservesStatefulCalls(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t, test.Golang(`package p
var n int
func next() bool { n++; return n == 2 }
func f() { if next() { println(1) } else if next() { println(2) } }`))
}

// Synthetic semantic counterexample from the Sonar safety audit; not corpus evidence.
func TestRemoveDuplicateConditionsPreservesInterveningCall(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t, test.Golang(`package p
var x int
func next() bool { x++; return false }
func f() { if x > 0 { println(1) } else if next() { println(2) } else if x > 0 { println(3) } }`))
}

// Synthetic semantic counterexample from the Sonar safety audit; not corpus evidence.
func TestRemoveDuplicateConditionsPreservesShadowedInit(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.RemoveDuplicateConditions{})
	spec.RewriteRun(t, test.Golang(`package p
func f(x int) { if x > 0 { println(1) } else if x := 2; x > 0 { println(x) } }`))
}
