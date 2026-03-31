/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindWeakHashMd5New(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindWeakHash{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "crypto/md5"

			func f() {
				h := md5.New()
				_ = h
			}
		`),
	)
}

func TestFindWeakHashSha1New(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindWeakHash{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "crypto/sha1"

			func f() {
				h := sha1.New()
				_ = h
			}
		`),
	)
}

func TestFindWeakHashNoChangeSha256(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindWeakHash{})
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
