/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyIfReturnBoolElseForm(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyIfReturnBool{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(cond bool) bool {
				if cond {
					return true
				} else {
					return false
				}
			}
		`, `
			package main

			func f(cond bool) bool {
				return cond
			}
		`),
	)
}

func TestSimplifyIfReturnBoolElseFormInverted(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyIfReturnBool{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(cond bool) bool {
				if cond {
					return false
				} else {
					return true
				}
			}
		`, `
			package main

			func f(cond bool) bool {
				return !cond
			}
		`),
	)
}

func TestSimplifyIfReturnBoolFollowingReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyIfReturnBool{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(cond bool) bool {
				if cond {
					return true
				}
				return false
			}
		`, `
			package main

			func f(cond bool) bool {
				return cond
			}
		`),
	)
}

func TestSimplifyIfReturnBoolFollowingReturnInverted(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyIfReturnBool{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(cond bool) bool {
				if cond {
					return false
				}
				return true
			}
		`, `
			package main

			func f(cond bool) bool {
				return !cond
			}
		`),
	)
}

func TestSimplifyIfReturnBoolNoChangeSameBool(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyIfReturnBool{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(cond bool) bool {
				if cond {
					return true
				}
				return true
			}
		`),
	)
}

func TestSimplifyIfReturnBoolNoChangeWithInit(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyIfReturnBool{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f() bool {
				if x := compute(); x {
					return true
				}
				return false
			}

			func compute() bool {
				return true
			}
		`),
	)
}

func TestSimplifyIfReturnBoolNoChangeNonBoolReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyIfReturnBool{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(cond bool) int {
				if cond {
					return 1
				}
				return 0
			}
		`),
	)
}

func TestSimplifyIfReturnBoolNoChangeExtraStatement(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.SimplifyIfReturnBool{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(cond bool) bool {
				if cond {
					println("hi")
					return true
				}
				return false
			}
		`),
	)
}
