/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindLongReceiverName(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindLongReceiverName{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type Foo struct{}

			func (self *Foo) Bar() {
			}
		`),
	)
}

func TestFindLongReceiverNameNoChangeShort(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindLongReceiverName{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type Foo struct{}

			func (f *Foo) Bar() {
			}
		`),
	)
}
