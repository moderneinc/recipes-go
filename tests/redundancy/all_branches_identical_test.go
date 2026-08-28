/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAllBranchesIdenticalSimpleIfElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.AllBranchesIdentical{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(cond bool) {
				if cond {
					println("hello")
				} else {
					println("hello")
				}
			}
		`, `
			package main

			func f(cond bool) {
				{
					println("hello")
				}
			}
		`),
	)
}

func TestAllBranchesIdenticalThreeBranches(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.AllBranchesIdentical{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b bool) {
				if a {
					println("hello")
				} else if b {
					println("hello")
				} else {
					println("hello")
				}
			}
		`, `
			package main

			func f(a, b bool) {
				{
					println("hello")
				}
			}
		`),
	)
}

func TestAllBranchesIdenticalConditionHasSideEffect(t *testing.T) {
	// given a bare call condition whose value is dead but whose evaluation
	// has a side effect, the call is hoisted before the body instead of dropped.
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.AllBranchesIdentical{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func consume() bool {
				println("side effect")
				return true
			}

			func f() {
				if consume() {
					println("hello")
				} else {
					println("hello")
				}
			}
		`, `
			package main

			func consume() bool {
				println("side effect")
				return true
			}

			func f() {
				consume()
				{
					println("hello")
				}
			}
		`),
	)
}

func TestAllBranchesIdenticalElseIfConditionHasSideEffect(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.AllBranchesIdentical{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func consume() bool {
				println("side effect")
				return true
			}

			func f(a bool) {
				if a {
					println("hello")
				} else if consume() {
					println("hello")
				} else {
					println("hello")
				}
			}
		`),
	)
}

func TestAllBranchesIdenticalConditionReceivesFromChannel(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.AllBranchesIdentical{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(ch chan bool) {
				if <-ch {
					println("hello")
				} else {
					println("hello")
				}
			}
		`),
	)
}

func TestAllBranchesIdenticalInitStatement(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.AllBranchesIdentical{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func lookup() (int, bool) {
				return 1, true
			}

			func f() {
				if v, ok := lookup(); ok {
					println(v)
				} else {
					println(v)
				}
			}
		`),
	)
}

func TestAllBranchesIdenticalNoElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.AllBranchesIdentical{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(cond bool) {
				if cond {
					println("hello")
				}
			}
		`),
	)
}

func TestAllBranchesIdenticalDifferentBodies(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.AllBranchesIdentical{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(cond bool) {
				if cond {
					println("hello")
				} else {
					println("world")
				}
			}
		`),
	)
}

func TestAllBranchesIdenticalElseIfDiffers(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.AllBranchesIdentical{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b bool) {
				if a {
					println("hello")
				} else if b {
					println("world")
				} else {
					println("hello")
				}
			}
		`),
	)
}
