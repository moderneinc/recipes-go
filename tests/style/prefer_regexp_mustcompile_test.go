/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferRegexpMustCompileCollapsesGuard(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferRegexpMustCompile{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "regexp"

			func f(p string) *regexp.Regexp {
				re, err := regexp.Compile(p)
				if err != nil {
					panic(err)
				}
				return re
			}
		`, `
			package main

			import "regexp"

			func f(p string) *regexp.Regexp {
				re := regexp.MustCompile(p)
				return re
			}
		`),
	)
}

func TestPreferRegexpMustCompileNoGuard(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferRegexpMustCompile{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "regexp"

			func f(p string) {
				re, err := regexp.Compile(p)
				_ = re
				_ = err
			}
		`),
	)
}

func TestPreferRegexpMustCompileErrUsedLater(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferRegexpMustCompile{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"fmt"
				"regexp"
			)

			func f(p string) *regexp.Regexp {
				re, err := regexp.Compile(p)
				if err != nil {
					return nil
				}
				fmt.Println(err)
				return re
			}
		`),
	)
}

func TestPreferRegexpMustCompileBlankError(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.PreferRegexpMustCompile{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "regexp"

			func f(p string) *regexp.Regexp {
				re, _ := regexp.Compile(p)
				return re
			}
		`),
	)
}
