/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/pkg/test"
)

func TestFindGetterPrefix(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindGetterPrefix{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type User struct{}

			func (u *User) GetName() string {
				return ""
			}
		`),
	)
}

func TestFindGetterPrefixNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindGetterPrefix{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type User struct{}

			func (u *User) Name() string {
				return ""
			}
		`),
	)
}

func TestFindGetterPrefixNoChangeFreeFunction(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.FindGetterPrefix{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func GetUser() string {
				return ""
			}
		`),
	)
}
