/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package expstd_test

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/expstd"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
)

func slogSpec() *test.RecipeSpec {
	return test.NewRecipeSpec().WithRecipe(&expstd.MigrateXExpSlogToStdlib{})
}

// The common case: a file on the non-context helpers is a plain path swap.
func TestSlogPlainSwap(t *testing.T) {
	slogSpec().RewriteRun(t,
		test.Golang(`
			package app

			import "golang.org/x/exp/slog"

			func run() {
				slog.Info("starting", "port", 8080)
				slog.Error("failed", "err", "boom")
			}
		`, `
			package app

			import "log/slog"

			func run() {
				slog.Info("starting", "port", 8080)
				slog.Error("failed", "err", "boom")
			}
		`),
	)
}

// The standard library never took the Ctx spelling, so those calls are renamed
// as the import moves.
func TestSlogRenamesCtxHelpers(t *testing.T) {
	slogSpec().RewriteRun(t,
		test.Golang(`
			package app

			import (
				"context"

				"golang.org/x/exp/slog"
			)

			func run(ctx context.Context) {
				slog.DebugCtx(ctx, "debugging")
				slog.InfoCtx(ctx, "starting", "port", 8080)
				slog.WarnCtx(ctx, "slow")
				slog.ErrorCtx(ctx, "failed", "err", "boom")
			}
		`, `
			package app

			import (
				"context"

				"log/slog"
			)

			func run(ctx context.Context) {
				slog.DebugContext(ctx, "debugging")
				slog.InfoContext(ctx, "starting", "port", 8080)
				slog.WarnContext(ctx, "slow")
				slog.ErrorContext(ctx, "failed", "err", "boom")
			}
		`),
	)
}

// The same helpers on a logger value, where the receiver's type is not the slog
// package. A leading context.Context stands in for the attribution x/exp cannot
// supply.
func TestSlogRenamesLoggerCtxHelpers(t *testing.T) {
	slogSpec().RewriteRun(t,
		test.Golang(`
			package app

			import (
				"context"

				"golang.org/x/exp/slog"
			)

			type service struct {
				logger *slog.Logger
			}

			func (s *service) handle(ctx context.Context) {
				s.logger.InfoCtx(ctx, "handling")
				s.logger.ErrorCtx(ctx, "failed")
			}
		`, `
			package app

			import (
				"context"

				"log/slog"
			)

			type service struct {
				logger *slog.Logger
			}

			func (s *service) handle(ctx context.Context) {
				s.logger.InfoContext(ctx, "handling")
				s.logger.ErrorContext(ctx, "failed")
			}
		`),
	)
}

// An unrelated InfoCtx that takes no context is not one of the renamed helpers.
func TestSlogLeavesUnrelatedCtxMethod(t *testing.T) {
	slogSpec().RewriteRun(t,
		test.Golang(`
			package app

			import "golang.org/x/exp/slog"

			type auditor struct{}

			func (auditor) InfoCtx(msg string) {}

			func run(a auditor) {
				a.InfoCtx("noted")
				slog.Info("starting")
			}
		`, `
			package app

			import "log/slog"

			type auditor struct{}

			func (auditor) InfoCtx(msg string) {}

			func run(a auditor) {
				a.InfoCtx("noted")
				slog.Info("starting")
			}
		`),
	)
}

// log/slog arrived in Go 1.21; an older module keeps x/exp.
func TestSlogBlockedByGoVersion(t *testing.T) {
	slogSpec().RewriteRun(t,
		test.GoProject("app",
			test.GoMod(`
				module example.com/app

				go 1.20

				require golang.org/x/exp v0.0.0-20230713183714-613f0c0eb8a1
			`),
			test.Golang(`
				package app

				import "golang.org/x/exp/slog"

				func run() {
					slog.Info("starting")
				}
			`),
		),
	)
}

// Both packages imported: they bind the same name, so the file is left alone.
func TestSlogBlockedWhenStdlibAlreadyImported(t *testing.T) {
	slogSpec().RewriteRun(t,
		test.Golang(`
			package app

			import (
				"log/slog"

				expslog "golang.org/x/exp/slog"
			)

			func run() {
				slog.Info("starting")
				expslog.Info("also starting")
			}
		`),
	)
}
