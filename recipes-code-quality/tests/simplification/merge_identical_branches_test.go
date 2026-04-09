/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestMergeIdenticalBranchesSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeIdenticalBranches{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b bool) {
				if a {
					println("x")
				} else if b {
					println("x")
				}
			}
		`, `
			package main

			func f(a, b bool) {
				if a || b {
					println("x")
				}
			}
		`),
	)
}

func TestMergeIdenticalBranchesWithElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeIdenticalBranches{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b bool) {
				if a {
					println("x")
				} else if b {
					println("x")
				} else {
					println("y")
				}
			}
		`, `
			package main

			func f(a, b bool) {
				if a || b {
					println("x")
				} else {
					println("y")
				}
			}
		`),
	)
}

func TestMergeIdenticalBranchesThreeWay(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeIdenticalBranches{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b, c bool) {
				if a {
					println("x")
				} else if b {
					println("x")
				} else if c {
					println("x")
				}
			}
		`, `
			package main

			func f(a, b, c bool) {
				if a || b || c {
					println("x")
				}
			}
		`),
	)
}

func TestMergeIdenticalBranchesNoChangeDifferentBodies(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeIdenticalBranches{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b bool) {
				if a {
					println("x")
				} else if b {
					println("y")
				}
			}
		`),
	)
}

func TestMergeIdenticalBranchesNoChangeNoElseIf(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeIdenticalBranches{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a bool) {
				if a {
					println("x")
				} else {
					println("y")
				}
			}
		`),
	)
}

func TestMergeIdenticalBranchesPartialMerge(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeIdenticalBranches{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b, c bool) {
				if a {
					println("x")
				} else if b {
					println("x")
				} else if c {
					println("y")
				}
			}
		`, `
			package main

			func f(a, b, c bool) {
				if a || b {
					println("x")
				} else if c {
					println("y")
				}
			}
		`),
	)
}
