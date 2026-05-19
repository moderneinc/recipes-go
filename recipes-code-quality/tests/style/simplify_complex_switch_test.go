/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyComplexSwitch(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.SimplifyComplexSwitch{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) string {
				switch x {
				case 1:
					return "a"
				case 2:
					return "b"
				case 3:
					return "c"
				case 4:
					return "d"
				case 5:
					return "e"
				case 6:
					return "f"
				case 7:
					return "g"
				case 8:
					return "h"
				case 9:
					return "i"
				case 10:
					return "j"
				case 11:
					return "k"
				case 12:
					return "l"
				}
				return ""
			}
		`),
	)
}

func TestSimplifyComplexSwitchNoChangeSmall(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.SimplifyComplexSwitch{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(x int) string {
				switch x {
				case 1:
					return "a"
				case 2:
					return "b"
				case 3:
					return "c"
				}
				return ""
			}
		`),
	)
}
