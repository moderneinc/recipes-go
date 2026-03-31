/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/style"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindTlsInsecureSkipVerifyTrue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTlsInsecureSkipVerify{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "crypto/tls"

			func f() *tls.Config {
				return &tls.Config{
					InsecureSkipVerify: true,
				}
			}
		`),
	)
}

func TestFindTlsInsecureSkipVerifyNoChangeFalse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.FindTlsInsecureSkipVerify{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "crypto/tls"

			func f() *tls.Config {
				return &tls.Config{
					InsecureSkipVerify: false,
				}
			}
		`),
	)
}
