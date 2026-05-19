/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestEnforceTlsVerificationTrue(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnforceTlsVerification{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "crypto/tls"

			func f() *tls.Config {
				return &tls.Config{
					InsecureSkipVerify: true,
				}
			}
		`, `
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

func TestEnforceTlsVerificationNoChangeFalse(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.EnforceTlsVerification{})
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
