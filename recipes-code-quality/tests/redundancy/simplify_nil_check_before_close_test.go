/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/redundancy"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestSimplifyNilCheckBeforeCloseSimple(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyNilCheckBeforeClose{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(f *os.File) {
				if f != nil {
					f.Close()
				}
			}
		`, `
			package main

			import "os"

			func f(f *os.File) {
				f.Close()
			}
		`),
	)
}

func TestSimplifyNilCheckBeforeCloseNilOnLeft(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyNilCheckBeforeClose{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f(f *os.File) {
				if nil != f {
					f.Close()
				}
			}
		`, `
			package main

			import "os"

			func f(f *os.File) {
				f.Close()
			}
		`),
	)
}

func TestSimplifyNilCheckBeforeCloseNoChangeNotClose(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyNilCheckBeforeClose{})
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

func TestSimplifyNilCheckBeforeCloseNoChangeDifferentVar(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyNilCheckBeforeClose{})
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

func TestSimplifyNilCheckBeforeCloseNoChangeMultipleStatements(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&redundancy.SimplifyNilCheckBeforeClose{})
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
