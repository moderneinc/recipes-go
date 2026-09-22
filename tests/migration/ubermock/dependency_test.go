/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package ubermock_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/ubermock"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func dependencySpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&ubermock.UpdateGolangMockDependency{})
}

// A migrated source tree: nothing imports the archived module, so the require is
// repointed in place. The composite runs FormatGoMod afterwards to restore
// module-path order.
func TestDependencyRepointsWhenSourceMigrated(t *testing.T) {
	dependencySpec().RewriteRun(t,
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
					go.uber.org/mock v0.6.0
					github.com/stretchr/testify v1.10.0
				)
			`),
			test.Golang(`
				package uc

				import "go.uber.org/mock/gomock"

				func ctrlFor(t gomock.TestReporter) *gomock.Controller {
					return gomock.NewController(t)
				}
			`),
		),
	)
}

// A standalone require directive is repointed the same way.
func TestDependencyStandaloneRequire(t *testing.T) {
	dependencySpec().RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/golang/mock v1.6.0
			`, `
				module example.com/app

				go 1.22

				require go.uber.org/mock v0.6.0
			`),
			test.Golang(`
				package uc

				import "go.uber.org/mock/gomock"

				func ctrlFor(t gomock.TestReporter) *gomock.Controller {
					return gomock.NewController(t)
				}
			`),
		),
	)
}

// An indirect requirement keeps its marker across the move.
func TestDependencyKeepsIndirectMarker(t *testing.T) {
	dependencySpec().RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/golang/mock v1.6.0 // indirect
				)
			`, `
				module example.com/app

				go 1.22

				require (
					go.uber.org/mock v0.6.0 // indirect
				)
			`),
			test.Golang(`
				package uc

				import "go.uber.org/mock/gomock"

				func ctrlFor(t gomock.TestReporter) *gomock.Controller {
					return gomock.NewController(t)
				}
			`),
		),
	)
}

// One file imports both forks, which the source swap refuses rather than
// duplicate an import, so it stays on the archived module and both are required.
func TestDependencyKeepsBothWhileOldStillImported(t *testing.T) {
	dependencySpec().RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require (
					github.com/golang/mock v1.6.0
				)
			`, `
				module example.com/app

				go 1.22

				require (
					github.com/golang/mock v1.6.0
					go.uber.org/mock v0.6.0
				)
			`),
			test.Golang(`
				package uc

				import "go.uber.org/mock/gomock"

				func ctrlFor(t gomock.TestReporter) *gomock.Controller {
					return gomock.NewController(t)
				}
			`),
			test.Golang(`
				package legacy

				import (
					oldmock "github.com/golang/mock/gomock"
					"go.uber.org/mock/gomock"
				)

				func ctrlFor(t gomock.TestReporter) *oldmock.Controller {
					return oldmock.NewController(t)
				}
			`),
		),
	)
}

// The requirement is repointed from source that has not been swapped yet, since
// the scan asks where each file will land rather than where it is. The Moderne
// CLI scans before a recipe list's edits are applied, so a go.mod decision read
// off the post-swap imports never fires there.
func TestDependencyRepointsAheadOfSourceMigration(t *testing.T) {
	dependencySpec().RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/golang/mock v1.6.0
			`, `
				module example.com/app

				go 1.22

				require go.uber.org/mock v0.6.0
			`),
			test.Golang(`
				package uc

				import "github.com/golang/mock/gomock"

				func ctrlFor(t gomock.TestReporter) *gomock.Controller {
					return gomock.NewController(t)
				}
			`),
		),
	)
}

// Nothing in the source touches either fork, so go.mod is untouched.
func TestDependencyNoChangeWithoutMockUsage(t *testing.T) {
	dependencySpec().RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.22

				require github.com/stretchr/testify v1.10.0
			`),
			test.Golang(`
				package uc

				import "testing"

				func TestThing(t *testing.T) {}
			`),
		),
	)
}
