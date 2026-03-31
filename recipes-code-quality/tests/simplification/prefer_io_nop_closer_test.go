/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferIoNopCloser(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferIoNopCloser{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"io"
				"io/ioutil"
				"strings"
			)

			func f() io.ReadCloser {
				return ioutil.NopCloser(strings.NewReader("hello"))
			}
		`, `
			package main

			import (
				"io"
				"io/ioutil"
				"strings"
			)

			func f() io.ReadCloser {
				return io.NopCloser(strings.NewReader("hello"))
			}
		`),
	)
}
