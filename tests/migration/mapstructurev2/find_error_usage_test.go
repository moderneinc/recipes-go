/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package mapstructurev2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/mapstructurev2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func findErrorSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&mapstructurev2.FindMapstructureErrorUsage{})
}

// Every route to the struct is reported, so the hand rework has the full list.
func TestFindErrorUsageMarksReferences(t *testing.T) {
	findErrorSpec().RewriteRun(t,
		test.Golang(`
			package profile

			import (
				"errors"

				"github.com/mitchellh/mapstructure"
			)

			func convert(err error) error {
				mapErr := &mapstructure.Error{}
				if !errors.As(err, &mapErr) {
					return err
				}
				return err
			}
		`, `
			package profile

			import (
				"errors"

				"github.com/mitchellh/mapstructure"
			)

			func convert(err error) error {
				mapErr := &/*~~(mapstructure.Error is not in go-viper/mapstructure/v2; the fork joins decode failures with errors.Join, so unwrap them with errors.Is/errors.As instead of reading Errors []string)~~>*/mapstructure.Error{}
				if !errors.As(err, &mapErr) {
					return err
				}
				return err
			}
		`),
	)
}

// A file using only migratable API has nothing to report.
func TestFindErrorUsageQuietOnMigratableFile(t *testing.T) {
	findErrorSpec().RewriteRun(t,
		test.Golang(`
			package awsssm

			import "github.com/mitchellh/mapstructure"

			func decode(in, out interface{}) error {
				return mapstructure.Decode(in, out)
			}
		`),
	)
}
