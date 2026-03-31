/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/simplification"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferIoDiscard(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferIoDiscard{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"io"
				"io/ioutil"
			)

			var _ io.Writer = ioutil.Discard
		`, `
			package main

			import (
				"io"
				"io/ioutil"
			)

			var _ io.Writer = io.Discard
		`),
	)
}
