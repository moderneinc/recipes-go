/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/code-quality/recipes/performance"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func TestFindJsonMarshalInForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.ReuseJsonCodecInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func f(items []string) {
				for _, item := range items {
					_, _ = json.Marshal(item)
				}
			}
		`),
	)
}

func TestFindJsonUnmarshalInClassicForLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.ReuseJsonCodecInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func f(data [][]byte) {
				for i := 0; i < len(data); i++ {
					var v interface{}
					_ = json.Unmarshal(data[i], &v)
				}
			}
		`),
	)
}

func TestFindJsonNoChangeOutsideLoop(t *testing.T) {
	spec := test.NewRecipeSpec().WithRecipe(&performance.ReuseJsonCodecInLoop{})
	spec.RewriteRun(t,
		test.Golang(`
			package main

			import "encoding/json"

			func f(item string) {
				_, _ = json.Marshal(item)
			}
		`),
	)
}
