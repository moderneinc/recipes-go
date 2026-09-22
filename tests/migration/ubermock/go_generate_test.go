/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package ubermock_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/ubermock"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func goGenerateSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&ubermock.UpdateMockgenGoGenerateDirectives{})
}

// appvia/terranetes-controller runs mockgen out of its vendor tree, so the
// module path appears mid-argument rather than at the start.
func TestGoGenerateVendoredMockgen(t *testing.T) {
	goGenerateSpec().RewriteRun(t,
		test.Golang(`
			package eks

			//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -package mocks -destination=mocks/ec2_zz.go github.com/appvia/terranetes-controller/pkg/utils/preload/eks EC2API
			//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -package mocks -destination=mocks/eks_zz.go github.com/appvia/terranetes-controller/pkg/utils/preload/eks EKSAPI

			type EC2API interface{}
		`, `
			package eks

			//go:generate go run ../../../../vendor/go.uber.org/mock/mockgen -package mocks -destination=mocks/ec2_zz.go github.com/appvia/terranetes-controller/pkg/utils/preload/eks EC2API
			//go:generate go run ../../../../vendor/go.uber.org/mock/mockgen -package mocks -destination=mocks/eks_zz.go github.com/appvia/terranetes-controller/pkg/utils/preload/eks EKSAPI

			type EC2API interface{}
		`),
	)
}

// A bare `mockgen` invocation names no module path, so there is nothing to move.
func TestGoGenerateBareMockgen(t *testing.T) {
	goGenerateSpec().RewriteRun(t,
		test.Golang(`
			package store

			//go:generate mockgen -source=store.go -destination=store_mock.go

			type Store interface{}
		`),
	)
}

// A plain comment naming the module is prose, not a directive, and is left alone.
func TestGoGenerateLeavesProseComments(t *testing.T) {
	goGenerateSpec().RewriteRun(t,
		test.Golang(`
			package store

			// Mocks in this package are generated with github.com/golang/mock.
			type Store interface{}
		`),
	)
}
