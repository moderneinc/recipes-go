/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package mapstructurev2_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/mapstructurev2"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// The whole module moves: imports and the go.mod require, with FormatGoMod
// restoring module-path order afterwards.
func TestMigrateWholeModule(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&mapstructurev2.MigrateToGoViperMapstructure{}).RewriteRun(t,
		test.GoProject("mole",
			test.GoMod(`
				module example.com/mole

				go 1.22

				require (
					github.com/mitchellh/mapstructure v1.5.0
					github.com/sirupsen/logrus v1.9.3
				)
			`, `
				module example.com/mole

				go 1.22

				require (
					github.com/go-viper/mapstructure/v2 v2.5.0
					github.com/sirupsen/logrus v1.9.3
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
			`, `
				package mole

				import "github.com/go-viper/mapstructure/v2"

				func toInstances(data interface{}) ([]string, error) {
					var instances []string
					err := mapstructure.Decode(data, &instances)
					return instances, err
				}
			`),
		),
	)
}

// One file names the dropped Error struct, so it stays on v1 and the module
// requires both until it is reworked by hand.
func TestMigrateKeepsBothWhenAFileIsBlocked(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&mapstructurev2.MigrateToGoViperMapstructure{}).RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/mitchellh/mapstructure v1.5.0
				)
			`, `
				module example.com/app

				go 1.22

				require (
					github.com/go-viper/mapstructure/v2 v2.5.0
					github.com/mitchellh/mapstructure v1.5.0
				)
			`),
			test.Golang(`
				package decode

				import "github.com/mitchellh/mapstructure"

				func plain(in, out interface{}) error {
					return mapstructure.Decode(in, out)
				}
			`, `
				package decode

				import "github.com/go-viper/mapstructure/v2"

				func plain(in, out interface{}) error {
					return mapstructure.Decode(in, out)
				}
			`),
			test.Golang(`
				package legacy

				import "github.com/mitchellh/mapstructure"

				func messages(err error) []string {
					msErr, ok := err.(*mapstructure.Error)
					if !ok {
						return nil
					}
					return msErr.Errors
				}
			`),
		),
	)
}
