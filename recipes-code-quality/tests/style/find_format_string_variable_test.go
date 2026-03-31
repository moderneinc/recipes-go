/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindFormatStringVariable(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindFormatStringVariable{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(userInput string) {
				_ = fmt.Sprintf(userInput)
			}
		`),
	)
}

func TestFindFormatStringVariableNoChangeLiteral(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindFormatStringVariable{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f(name string) {
				_ = fmt.Sprintf("hello %s", name)
			}
		`),
	)
}
