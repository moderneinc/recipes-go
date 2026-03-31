/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindExecCommand(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindExecCommand{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os/exec"

			func f() {
				exec.Command("ls", "-la")
			}
		`),
	)
}

func TestFindExecCommandNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindExecCommand{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() {
				fmt.Println("command")
			}
		`),
	)
}
