/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package ubermock_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/ubermock"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

// The composite over a module shaped like appvia/terranetes-controller: source
// imports, a go:generate directive and the go.mod require all move together, and
// FormatGoMod puts the repointed require back in module-path order.
func TestMigrateToUberMockWholeModule(t *testing.T) {
	test.NewRecipeSpec().WithRecipe(&ubermock.MigrateToUberMock{}).RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/golang/mock v1.6.0
					github.com/stretchr/testify v1.10.0
				)
			`, `
				module example.com/app

				go 1.22

				require (
					github.com/stretchr/testify v1.10.0
					go.uber.org/mock v0.6.0
				)
			`),
			test.Golang(`
				package eks

				import (
					"testing"

					"github.com/golang/mock/gomock"
				)

				//go:generate go run github.com/golang/mock/mockgen -package mocks -destination=mocks/ec2_zz.go example.com/app/eks EC2API

				func TestLoader(t *testing.T) {
					ctrl := gomock.NewController(t)
					defer ctrl.Finish()
					_ = ctrl
				}
			`, `
				package eks

				import (
					"testing"

					"go.uber.org/mock/gomock"
				)

				//go:generate go run go.uber.org/mock/mockgen -package mocks -destination=mocks/ec2_zz.go example.com/app/eks EC2API

				func TestLoader(t *testing.T) {
					ctrl := gomock.NewController(t)
					defer ctrl.Finish()
					_ = ctrl
				}
			`),
		),
	)
}
