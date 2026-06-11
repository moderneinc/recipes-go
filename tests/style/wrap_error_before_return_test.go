/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/style"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestWrapErrorBeforeReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.WrapErrorBeforeReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func doWork() ([]byte, error) {
				_, err := fmt.Println("work")
				if err != nil {
					return nil, err
				}
				return []byte("ok"), nil
			}
		`, `
			package main

			import "fmt"

			func doWork() ([]byte, error) {
				_, err := fmt.Println("work")
				if err != nil {
					return nil, fmt.Errorf("doWork: %w", err)
				}
				return []byte("ok"), nil
			}
		`),
	)
}

func TestWrapErrorBeforeReturnNoChangeAlreadyWrapped(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.WrapErrorBeforeReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "fmt"

			func doWork() ([]byte, error) {
				_, err := fmt.Println("work")
				if err != nil {
					return nil, fmt.Errorf("doWork: %w", err)
				}
				return []byte("ok"), nil
			}
		`),
	)
}

func TestWrapErrorBeforeReturnNoChangeSingleReturn(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&style.WrapErrorBeforeReturn{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			func doWork() error {
				return nil
			}
		`),
	)
}
