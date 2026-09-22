/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package expstd

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// The entry point: everything golang.org/x/exp contributed to the standard
// library, plus the go.mod cleanup.
//
// It is one scanning recipe rather than a RecipeList over the ones it composes,
// which all of them need anyway: every rewrite here is gated on the module's `go`
// directive, and the go.mod cleanup on what the rewrites leave behind, so a
// single scan answers the lot. Composing them would also be one scanning
// sub-recipe too many — the Moderne CLI hangs on a recipe list holding more than
// one.
type MigrateXExpToStdlib struct {
	recipe.ScanningBase
}

func (r *MigrateXExpToStdlib) Name() string {
	return "org.openrewrite.golang.migration.MigrateXExpToStdlib"
}
func (r *MigrateXExpToStdlib) DisplayName() string {
	return "Migrate `golang.org/x/exp` to the standard library"
}
func (r *MigrateXExpToStdlib) Description() string {
	return "Migrate `golang.org/x/exp/slices`, `golang.org/x/exp/maps`, `golang.org/x/exp/constraints` and `golang.org/x/exp/slog` to the `slices`, `maps`, `cmp` and `log/slog` packages that absorbed them, and drop the `golang.org/x/exp` requirement once nothing needs it. Each rewrite is gated on the module's `go` directive, and a file using API the standard library never took — the numeric constraints, or the pre-2023 boolean comparators — is left for review. Run `go mod tidy` afterwards to sync go.sum."
}
func (r *MigrateXExpToStdlib) Tags() []string {
	return []string{"migration", "stdlib", "slices", "maps", "slog"}
}

func (r *MigrateXExpToStdlib) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "exptostd", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *MigrateXExpToStdlib) InitialValue(*recipe.ExecutionContext) any {
	return &expUsageAcc{}
}
func (r *MigrateXExpToStdlib) Scanner(acc any) recipe.TreeVisitor {
	return visitor.Init(&expUsageScanner{acc: acc.(*expUsageAcc)})
}
func (r *MigrateXExpToStdlib) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&migrateXExpEditor{acc: acc.(*expUsageAcc)})
}

type migrateXExpEditor struct {
	visitor.GoVisitor
	acc *expUsageAcc
}

func (v *migrateXExpEditor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	return applyXExpMigrations(cu, &v.acc.moduleAcc)
}

func (v *migrateXExpEditor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	if !v.acc.migrationEmptiesEveryFile() {
		return gm
	}
	return migration.RemoveRequire(gm, expModule)
}
