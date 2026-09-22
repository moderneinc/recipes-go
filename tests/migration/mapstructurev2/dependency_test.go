/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package mapstructurev2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/mapstructurev2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func dependencySpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&mapstructurev2.UpdateMapstructureDependency{})
}

// davrodpin/mole's shape: the requirement is repointed from source that has not
// been swapped yet, since the scan asks where each file will land rather than
// where it is.
func TestDependencyRepointsAheadOfSourceMigration(t *testing.T) {
	dependencySpec().RewriteRun(t,
		test.GoProject("mole",
			test.GoMod(`
				module example.com/mole

				go 1.22

				require (
					github.com/gofrs/uuid v4.4.0+incompatible
					github.com/mitchellh/mapstructure v1.4.1
				)
			`, `
				module example.com/mole

				go 1.22

				require (
					github.com/gofrs/uuid v4.4.0+incompatible
					github.com/go-viper/mapstructure/v2 v2.5.0
				)
			`),
			test.Golang(`
				package mole

				import "github.com/mitchellh/mapstructure"

				func toInstances(data interface{}) ([]string, error) {
					var instances []string
					err := mapstructure.Decode(data, &instances)
					return instances, err
				}
			`),
		),
	)
}
