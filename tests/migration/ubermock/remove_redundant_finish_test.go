/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package ubermock_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/ubermock"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func finishSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&ubermock.RemoveRedundantGomockFinish{})
}

// err0r500/go-realworld-clean's standard opening two lines: NewController
// registers the finish through t.Cleanup, so the defer only repeats it.
func TestRemoveFinishAfterNewControllerFromT(t *testing.T) {
	finishSpec().RewriteRun(t,
		test.Golang(`
			package uc

			import (
				"testing"

				"go.uber.org/mock/gomock"
			)

			func TestTags(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()

				i := newInteractor(mockCtrl)
				i.TagRW.EXPECT().GetAll().Return(nil, nil)
			}
		`, `
			package uc

			import (
				"testing"

				"go.uber.org/mock/gomock"
			)

			func TestTags(t *testing.T) {
				mockCtrl := gomock.NewController(t)

				i := newInteractor(mockCtrl)
				i.TagRW.EXPECT().GetAll().Return(nil, nil)
			}
		`),
	)
}

// The archived library has registered the cleanup since v1.5.0, so the defer is
// redundant before the path swap as well.
func TestRemoveFinishOnArchivedImport(t *testing.T) {
	finishSpec().RewriteRun(t,
		test.Golang(`
			package uc

			import (
				"testing"

				"github.com/golang/mock/gomock"
			)

			func TestTags(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()
				_ = ctrl
			}
		`, `
			package uc

			import (
				"testing"

				"github.com/golang/mock/gomock"
			)

			func TestTags(t *testing.T) {
				ctrl := gomock.NewController(t)
				_ = ctrl
			}
		`),
	)
}

// A controller built from something other than a testing type has no Cleanup to
// hook, so its Finish is doing real work.
func TestRemoveFinishKeepsCustomReporter(t *testing.T) {
	finishSpec().RewriteRun(t,
		test.Golang(`
			package uc

			import "go.uber.org/mock/gomock"

			type reporter struct{}

			func (reporter) Errorf(format string, args ...any) {}
			func (reporter) Fatalf(format string, args ...any) {}

			func run() {
				ctrl := gomock.NewController(reporter{})
				defer ctrl.Finish()
				_ = ctrl
			}
		`),
	)
}

// A Finish on a value that is not a gomock controller is untouched, even when it
// shares a name with one in another function.
func TestRemoveFinishLeavesUnrelatedFinish(t *testing.T) {
	finishSpec().RewriteRun(t,
		test.Golang(`
			package uc

			import (
				"testing"

				"go.uber.org/mock/gomock"
			)

			type tracer struct{}

			func (tracer) Finish() {}

			func newTracer() tracer { return tracer{} }

			func TestOne(t *testing.T) {
				ctrl := gomock.NewController(t)
				_ = ctrl
			}

			func TestTwo(t *testing.T) {
				ctrl := newTracer()
				defer ctrl.Finish()
			}
		`),
	)
}

// No gomock import at all: nothing to consider.
func TestRemoveFinishNoGomockImport(t *testing.T) {
	finishSpec().RewriteRun(t,
		test.Golang(`
			package uc

			import "testing"

			func TestThing(t *testing.T) {}
		`),
	)
}

// equinor/radix-log-api and blackPavlin/shop build the controller from a testify
// suite's T(), which hands over the running *testing.T — so the cleanup is
// registered and the deferred Finish is redundant just the same.
func TestRemoveFinishFromTestifySuiteT(t *testing.T) {
	finishSpec().RewriteRun(t,
		test.Golang(`
			package router

			import (
				"github.com/stretchr/testify/suite"
				"go.uber.org/mock/gomock"
			)

			type authnTestSuite struct {
				suite.Suite
			}

			func (s *authnTestSuite) TestAuthn() {
				ctrl := gomock.NewController(s.T())
				defer ctrl.Finish()

				_ = ctrl
			}
		`, `
			package router

			import (
				"github.com/stretchr/testify/suite"
				"go.uber.org/mock/gomock"
			)

			type authnTestSuite struct {
				suite.Suite
			}

			func (s *authnTestSuite) TestAuthn() {
				ctrl := gomock.NewController(s.T())

				_ = ctrl
			}
		`),
	)
}

// A controller assigned to a struct field rather than a local is not tracked,
// so nothing is dropped. JosiahWitt/ensure stores one that way.
func TestRemoveFinishLeavesFieldHeldController(t *testing.T) {
	finishSpec().RewriteRun(t,
		test.Golang(`
			package testutilx

			import (
				"github.com/stretchr/testify/suite"
				"go.uber.org/mock/gomock"
			)

			type Suite struct {
				suite.Suite
				Ctrl *gomock.Controller
			}

			func (s *Suite) SetupTest() {
				s.Ctrl = gomock.NewController(s.T())
			}

			func (s *Suite) TearDownTest() {
				defer s.Ctrl.Finish()
			}
		`),
	)
}
