/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseStrongHashMd5New(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseStrongHash{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "crypto/md5"

			func f() {
				h := md5.New()
				_ = h
			}
		`, `
			package main

			import (
				"crypto/sha256"
			)

			func f() {
				h := sha256.New()
				_ = h
			}
		`),
	)
}

// md5.Sum is intentionally left alone: its result type differs from sha256.Sum256.
func TestUseStrongHashNoChangeMd5Sum(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseStrongHash{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "crypto/md5"

			func f(data []byte) {
				h := md5.Sum(data)
				_ = h
			}
		`),
	)
}

func TestUseStrongHashSha1New(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseStrongHash{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "crypto/sha1"

			func f() {
				h := sha1.New()
				_ = h
			}
		`, `
			package main

			import (
				"crypto/sha256"
			)

			func f() {
				h := sha256.New()
				_ = h
			}
		`),
	)
}

// sha1.Sum is intentionally left alone: its result type differs from sha256.Sum256.
func TestUseStrongHashNoChangeSha1Sum(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseStrongHash{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "crypto/sha1"

			func f(data []byte) {
				h := sha1.Sum(data)
				_ = h
			}
		`),
	)
}

func TestUseStrongHashNoChangeSha256(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.UseStrongHash{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "crypto/sha256"

			func f() {
				h := sha256.New()
				_ = h
			}
		`),
	)
}
