/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestAvoidTimeSleep(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidTimeSleep{})
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

func TestAvoidTimeSleepDuration(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidTimeSleep{})
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

func TestAvoidTimeSleepNoChangeTimeNow(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidTimeSleep{})
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

func TestAvoidTimeSleepNoChangeOtherPkg(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.AvoidTimeSleep{})
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
