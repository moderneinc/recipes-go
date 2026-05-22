/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAuditExecCommand(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditExecCommand{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os/exec"

			func f() {
				exec.Command("ls", "-la")
			}
		`, `
			package main

			import "os/exec"

			func f() {/*~~(exec.Command call; ensure arguments are not from untrusted input)~~>*/
				exec.Command("ls", "-la")
			}
		`),
	)
}

func TestAuditExecCommandNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AuditExecCommand{})
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
