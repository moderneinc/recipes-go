/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestMergeCollapsibleIfBasic(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeCollapsibleIf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b bool) {
				if a {
					if b {
						println("both")
					}
				}
			}
		`, `
			package main

			func f(a, b bool) {
				if a && b {
					println("both")
				}
			}
		`),
	)
}

func TestMergeCollapsibleIfComparisonOperators(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeCollapsibleIf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x, y int) {
				if x > 0 {
					if y < 10 {
						println("in range")
					}
				}
			}
		`, `
			package main

			func f(x, y int) {
				if x > 0 && y < 10 {
					println("in range")
				}
			}
		`),
	)
}

func TestMergeCollapsibleIfWrapsOuterOr(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeCollapsibleIf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b, c bool) {
				if a || b {
					if c {
						println("ok")
					}
				}
			}
		`, `
			package main

			func f(a, b, c bool) {
				if (a || b) && c {
					println("ok")
				}
			}
		`),
	)
}

func TestMergeCollapsibleIfWrapsInnerOr(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeCollapsibleIf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b, c bool) {
				if a {
					if b || c {
						println("ok")
					}
				}
			}
		`, `
			package main

			func f(a, b, c bool) {
				if a && (b || c) {
					println("ok")
				}
			}
		`),
	)
}

func TestMergeCollapsibleIfWrapsBothOr(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeCollapsibleIf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b, c, d bool) {
				if a || b {
					if c || d {
						println("ok")
					}
				}
			}
		`, `
			package main

			func f(a, b, c, d bool) {
				if (a || b) && (c || d) {
					println("ok")
				}
			}
		`),
	)
}

func TestMergeCollapsibleIfNoChangeOuterElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeCollapsibleIf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b bool) {
				if a {
					if b {
						println("both")
					}
				} else {
					println("not a")
				}
			}
		`),
	)
}

func TestMergeCollapsibleIfNoChangeInnerElse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeCollapsibleIf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b bool) {
				if a {
					if b {
						println("both")
					} else {
						println("not b")
					}
				}
			}
		`),
	)
}

func TestMergeCollapsibleIfNoChangeMultipleStatements(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeCollapsibleIf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b bool) {
				if a {
					println("a")
					if b {
						println("both")
					}
				}
			}
		`),
	)
}

func TestMergeCollapsibleIfAndConditionsNotWrapped(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.MergeCollapsibleIf{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(a, b, c, d bool) {
				if a && b {
					if c && d {
						println("all")
					}
				}
			}
		`, `
			package main

			func f(a, b, c, d bool) {
				if a && b && c && d {
					println("all")
				}
			}
		`),
	)
}
