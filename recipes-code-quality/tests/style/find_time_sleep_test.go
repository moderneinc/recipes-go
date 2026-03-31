/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindTimeSleep(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTimeSleep{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f() {
				time.Sleep(time.Second)
			}
		`),
	)
}

func TestFindTimeSleepDuration(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTimeSleep{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f() {
				time.Sleep(100 * time.Millisecond)
			}
		`),
	)
}

func TestFindTimeSleepNoChangeTimeNow(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTimeSleep{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "time"

			func f() {
				_ = time.Now()
			}
		`),
	)
}

func TestFindTimeSleepNoChangeOtherPkg(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTimeSleep{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func f() {
				fmt.Println("hello")
			}
		`),
	)
}
