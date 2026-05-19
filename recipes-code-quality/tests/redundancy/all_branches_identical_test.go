/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/redundancy"
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
