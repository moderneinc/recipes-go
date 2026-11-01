/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package expstd

import (
	"strings"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Drops the x/exp requirement once the migration has emptied it.
type RemoveXExpDependency struct {
	recipe.ScanningBase
}

func (r *RemoveXExpDependency) Name() string {
	return "org.openrewrite.golang.migration.RemoveXExpDependency"
}
func (r *RemoveXExpDependency) DisplayName() string {
	return "Remove the `golang.org/x/exp` requirement once unused"
}
func (r *RemoveXExpDependency) Description() string {
	return "Drop the direct `require golang.org/x/exp` directive from go.mod once no source file needs it — either because nothing imports it any more, or because the migration moves every import that remains. x/exp holds far more than the packages these recipes cover, so an import of any other one keeps the requirement, as does an `// indirect` entry. Does not touch go.sum, so a `go mod tidy` is still needed."
}
func (r *RemoveXExpDependency) Tags() []string {
	return []string{"migration", "stdlib", "gomod"}
}

func (r *RemoveXExpDependency) InitialValue(*recipe.ExecutionContext) any {
	return &expUsageAcc{}
}
func (r *RemoveXExpDependency) Scanner(acc any) recipe.TreeVisitor {
	return visitor.Init(&expUsageScanner{acc: acc.(*expUsageAcc)})
}
func (r *RemoveXExpDependency) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&removeExpRequireEditor{acc: acc.(*expUsageAcc)})
}

// expUsageAcc holds a verdict on each file that imports x/exp for every go
// version the migrations are gated on. Whether the migration takes a file
// depends on the module's `go` directive, which another file carries, and the
// scan sees files in no guaranteed order — so the edit phase reads the verdict
// for the version the go.mod names, once both are known.
type expUsageAcc struct {
	moduleAcc
	// emptiedFrom holds, for each file importing x/exp, the oldest of
	// gateVersions from which the migrations leave it no x/exp import, or ""
	// when none does.
	emptiedFrom []string
}

type expUsageScanner struct {
	visitor.GoVisitor
	acc *expUsageAcc
}

func (v *expUsageScanner) VisitGoModDirective(d *golang.GoModDirective, p any) java.Tree {
	if d.Keyword == "go" && len(d.Values) == 1 {
		v.acc.goVersion = d.Values[0].Text
	}
	return d
}

func (v *expUsageScanner) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	if importsXExp(cu) {
		v.acc.emptiedFrom = append(v.acc.emptiedFrom, emptiedFrom(cu))
	}
	return cu
}

// emptiedFrom is the oldest of gateVersions from which the package migrations
// take every x/exp import out of cu, or "" when none does. A newer version only
// opens more of the migrations, so a file emptied at one version stays emptied
// at every later one.
func emptiedFrom(cu *golang.CompilationUnit) string {
	for _, version := range gateVersions {
		if !importsXExp(applyXExpMigrations(cu, &moduleAcc{goVersion: version})) {
			return version
		}
	}
	return ""
}

type removeExpRequireEditor struct {
	visitor.GoVisitor
	acc *expUsageAcc
}

func (v *removeExpRequireEditor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	if !v.acc.migrationEmptiesEveryFile() {
		return gm
	}
	return migration.RemoveRequire(gm, expModule)
}

// migrationEmptiesEveryFile reports whether the package migrations leave
// no x/exp import anywhere in the module. Asking it of the rewritten files
// answers the same, so the verdict does not depend on whether the source rewrite
// has run yet.
func (a *expUsageAcc) migrationEmptiesEveryFile() bool {
	for _, from := range a.emptiedFrom {
		if from == "" || !a.atLeast(from) {
			return false
		}
	}
	return true
}

// applyXExpMigrations runs the package migrations over cu in turn,
// draining the imports each queues so the next one sees them.
func applyXExpMigrations(cu *golang.CompilationUnit, acc *moduleAcc) *golang.CompilationUnit {
	editors := []recipe.TreeVisitor{
		(&MigrateXExpSlicesToStdlib{}).EditorWithData(acc),
		(&MigrateXExpMapsToStdlib{}).EditorWithData(acc),
		(&MigrateXExpConstraintsToStdlib{}).EditorWithData(acc),
		(&MigrateXExpSlogToStdlib{}).EditorWithData(acc),
	}
	ctx := recipe.NewExecutionContext()
	current := cu
	for _, editor := range editors {
		if migrated, ok := visitor.DrainAfterVisits(editor, editor.Visit(current, ctx), ctx).(*golang.CompilationUnit); ok {
			current = migrated
		}
	}
	return current
}

// importsXExp reports whether cu imports anything under golang.org/x/exp.
func importsXExp(cu *golang.CompilationUnit) bool {
	if cu == nil || cu.Imports == nil {
		return false
	}
	for _, rp := range cu.Imports.Elements {
		if path := rp.Element.Path(); path == expModule || strings.HasPrefix(path, expModule+"/") {
			return true
		}
	}
	return false
}
