/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/errorhandling"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestPreferErrorsIsNetClosedEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsNetClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net"

			func f(err error) bool {
				return err == net.ErrClosed
			}
		`, `
			package main

			import "net"

			func f(err error) bool {
				return errors.Is(err, net.ErrClosed)
			}
		`),
	)
}

func TestPreferErrorsIsNetClosedNotEqual(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsNetClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "net"

			func f(err error) bool {
				return err != net.ErrClosed
			}
		`, `
			package main

			import "net"

			func f(err error) bool {
				return !errors.Is(err, net.ErrClosed)
			}
		`),
	)
}

func TestPreferErrorsIsNetClosedNoChangeNil(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&errorhandling.PreferErrorsIsNetClosed{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func f(err error) bool {
				return err == nil
			}
		`),
	)
}
