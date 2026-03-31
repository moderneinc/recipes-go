/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindNilCheckBeforeCloseSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindNilCheckBeforeClose{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(f *os.File) {
				if f != nil {
					f.Close()
				}
			}
		`),
	)
}

func TestFindNilCheckBeforeCloseNilOnLeft(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindNilCheckBeforeClose{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(f *os.File) {
				if nil != f {
					f.Close()
				}
			}
		`),
	)
}

func TestFindNilCheckBeforeCloseNoChangeNotClose(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindNilCheckBeforeClose{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(r interface{ Read([]byte) int }) {
				buf := make([]byte, 10)
				if r != nil {
					r.Read(buf)
				}
			}
		`),
	)
}

func TestFindNilCheckBeforeCloseNoChangeDifferentVar(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindNilCheckBeforeClose{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(a *os.File, b *os.File) {
				if a != nil {
					b.Close()
				}
			}
		`),
	)
}

func TestFindNilCheckBeforeCloseNoChangeMultipleStatements(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.FindNilCheckBeforeClose{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(f *os.File) {
				if f != nil {
					f.Sync()
					f.Close()
				}
			}
		`),
	)
}
