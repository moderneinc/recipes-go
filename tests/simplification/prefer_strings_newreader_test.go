/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/simplification"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestPreferStringsNewReader(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsNewReader{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"bytes"
				"io"
			)

			func f(s string) io.Reader {
				return bytes.NewReader([]byte(s))
			}
		`, `
			package main

			import (
				"io"
				"strings"
			)

			func f(s string) io.Reader {
				return strings.NewReader(s)
			}
		`),
	)
}

// Skips a []byte argument, where strings.NewReader's string parameter would not compile.
func TestPreferStringsNewReaderNoChangeByteSliceArg(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsNewReader{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) {
				r := bytes.NewReader([]byte(b))
				_ = r
			}
		`),
	)
}

// A string literal is a string, so the rewrite still proceeds.
func TestPreferStringsNewReaderStringLiteralArg(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsNewReader{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"bytes"
				"io"
			)

			func f() io.Reader {
				return bytes.NewReader([]byte("hello"))
			}
		`, `
			package main

			import (
				"io"
				"strings"
			)

			func f() io.Reader {
				return strings.NewReader("hello")
			}
		`),
	)
}

// Skips a *bytes.Reader variable declaration, which *strings.Reader would not satisfy.
func TestPreferStringsNewReaderNoChangeTypedVarDecl(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsNewReader{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(s string) {
				var r *bytes.Reader = bytes.NewReader([]byte(s))
				_ = r
			}
		`),
	)
}

// An interface-typed declaration accepts both readers, so the rewrite proceeds.
func TestPreferStringsNewReaderInterfaceVarDecl(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsNewReader{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import (
				"bytes"
				"io"
			)

			func f(s string) {
				var r io.Reader = bytes.NewReader([]byte(s))
				_ = r
			}
		`, `
			package main

			import (
				"io"
				"strings"
			)

			func f(s string) {
				var r io.Reader = strings.NewReader(s)
				_ = r
			}
		`),
	)
}

func TestPreferStringsNewReaderNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsNewReader{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(b []byte) *bytes.Reader {
				return bytes.NewReader(b)
			}
		`),
	)
}

// Skips a direct return of *bytes.Reader, where *strings.Reader would not compile.
func TestPreferStringsNewReaderNoChangeBytesReaderContext(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&simplification.PreferStringsNewReader{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "bytes"

			func f(s string) *bytes.Reader {
				return bytes.NewReader([]byte(s))
			}
		`),
	)
}
