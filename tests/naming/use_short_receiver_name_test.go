/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/naming"
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

func TestUseShortReceiverNameNoChangeParameterCollision(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseShortReceiverName{})
	spec.RewriteRun(t,
		test.Golang(`
			package recv

			import (
				"encoding/xml"
			)

			type ExpirationDays int

			func (eDays ExpirationDays) MarshalXML(e *xml.Encoder, startElement xml.StartElement) error {
				if eDays == 0 {
					return nil
				}
				return e.EncodeElement(int(eDays), startElement)
			}
		`),
	)
}

func TestUseShortReceiverNameNoChangeNamedResultCollision(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseShortReceiverName{})
	spec.RewriteRun(t,
		test.Golang(`
			package recv

			type Foo struct{ n int }

			func (foo *Foo) Bar() (f int, err error) {
				f = foo.n
				return f, nil
			}
		`),
	)
}

func TestUseShortReceiverNameNoChangeShortVarDeclCollision(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseShortReceiverName{})
	spec.RewriteRun(t,
		test.Golang(`
			package recv

			type Foo struct{ n int }

			func (foo *Foo) Bar() int {
				f := foo.n
				return f
			}
		`),
	)
}

func TestUseShortReceiverNameNoChangeVarDeclCollision(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseShortReceiverName{})
	spec.RewriteRun(t,
		test.Golang(`
			package recv

			type Foo struct{ n int }

			func (foo *Foo) Bar() int {
				var f int
				f = foo.n
				return f
			}
		`),
	)
}

func TestUseShortReceiverNameNoChangeRangeVariableCollision(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseShortReceiverName{})
	spec.RewriteRun(t,
		test.Golang(`
			package recv

			type Foo struct{ items []int }

			func (foo *Foo) Sum() int {
				total := 0
				for _, f := range foo.items {
					total += f
				}
				return total
			}
		`),
	)
}

func TestUseShortReceiverNameNoChangeTypeSwitchCollision(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseShortReceiverName{})
	spec.RewriteRun(t,
		test.Golang(`
			package recv

			type Foo struct{ v any }

			func (foo *Foo) Kind() string {
				switch f := foo.v.(type) {
				case int:
					_ = f
					return "int"
				}
				return ""
			}
		`),
	)
}

func TestUseShortReceiverNameNoChangeTypeNameCollision(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseShortReceiverName{})
	spec.RewriteRun(t,
		test.Golang(`
			package recv

			type f struct{ n int }

			func (elem f) Twice() f {
				return f{n: elem.n * 2}
			}
		`),
	)
}

func TestUseShortReceiverNameShortensWhenNothingCollides(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.UseShortReceiverName{})
	spec.RewriteRun(t,
		test.Golang(`
			package recv

			type BoolWithInverseFlag struct{ name string }

			func (bif *BoolWithInverseFlag) String() string {
				return bif.name
			}
		`, `
			package recv

			type BoolWithInverseFlag struct{ name string }

			func (b *BoolWithInverseFlag) String() string {
				return b.name
			}
		`),
	)
}
