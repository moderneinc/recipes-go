/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes-code-quality/recipes/naming"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestRemoveGetterPrefix(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.RemoveGetterPrefix{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			type User struct{}

			func (u *User) GetName() string {
				return ""
			}
		`, `
			package main

			type User struct{}

			func (u *User) Name() string {
				return ""
			}
		`),
	)
}

func TestRemoveGetterPrefixNoChange(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.RemoveGetterPrefix{})
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

func TestRemoveGetterPrefixNoChangeFreeFunction(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&naming.RemoveGetterPrefix{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func GetUser() string {
				return ""
			}
		`),
	)
}
