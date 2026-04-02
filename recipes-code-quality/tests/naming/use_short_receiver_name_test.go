/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestUseShortReceiverName(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseShortReceiverName{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type Foo struct{}

			func (self *Foo) Bar() {
			}
		`, `
			package main

			type Foo struct{}

			func (f *Foo) Bar() {
			}
		`),
	)
}

func TestUseShortReceiverNameNoChangeShort(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseShortReceiverName{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type Foo struct{}

			func (f *Foo) Bar() {
			}
		`),
	)
}
