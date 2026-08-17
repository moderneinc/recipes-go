/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnsureFileClosed(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureFileClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				f, err := os.Open("file")
				_ = err
				_ = f
			}
		`, `
			package main

			import "os"

			func f() {
				f, err := os.Open("file")
				defer f.Close()
				_ = err
				_ = f
			}
		`),
	)
}

func TestEnsureFileClosedCreate(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureFileClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				f, err := os.Create("file")
				_ = err
				_ = f
			}
		`, `
			package main

			import "os"

			func f() {
				f, err := os.Create("file")
				defer f.Close()
				_ = err
				_ = f
			}
		`),
	)
}

func TestEnsureFileClosedAfterErrorCheck(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureFileClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() error {
				f, err := os.Open("file")
				if err != nil {
					return err
				}
				_ = f
				return nil
			}
		`, `
			package main

			import "os"

			func f() error {
				f, err := os.Open("file")
				if err != nil {
					return err
				}
				defer f.Close()
				_ = f
				return nil
			}
		`),
	)
}

func TestEnsureFileClosedTwoFilesInOneBlock(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureFileClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() error {
				src, err := os.Open("a")
				if err != nil {
					return err
				}
				dst, err := os.Create("b")
				_ = err
				_, _ = src, dst
				return nil
			}
		`, `
			package main

			import "os"

			func f() error {
				src, err := os.Open("a")
				if err != nil {
					return err
				}
				defer src.Close()
				dst, err := os.Create("b")
				defer dst.Close()
				_ = err
				_, _ = src, dst
				return nil
			}
		`),
	)
}

func TestEnsureFileClosedNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureFileClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				os.Getenv("X")
			}
		`),
	)
}

func TestEnsureFileClosedAlreadyDeferred(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnsureFileClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "os"

			func f() {
				f, err := os.Open("file")
				defer f.Close()
				_ = err
			}
		`),
	)
}
