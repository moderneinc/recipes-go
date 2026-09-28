/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"fmt"
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/preconditions"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// PartiallyResolvedGoModRow is one row of the FindPartiallyResolvedGoMod data table.
type PartiallyResolvedGoModRow struct {
	SourcePath        string
	ModulePath        string
	ResolutionStatus  string
	UnresolvedImports string
	ResolutionError   string
}

var partiallyResolvedGoModTable = recipe.NewDataTable[PartiallyResolvedGoModRow](
	"org.openrewrite.golang.migration.table.PartiallyResolvedGoMod",
	"go.mod files not fully resolved offline",
	"Every go.mod whose module graph did not fully resolve at parse time, so GoModTidy skipped adding and removing requirements. The offline no-op is safe but silent; this table names the affected modules and why resolution stopped short so a run can be triaged and re-run once the modules resolve.",
	[]recipe.ColumnDescriptor{
		{Name: "sourcePath", DisplayName: "Source path", Description: "The go.mod that could not be fully resolved.", Type: "String"},
		{Name: "modulePath", DisplayName: "Module path", Description: "The module the go.mod declares.", Type: "String"},
		{Name: "resolutionStatus", DisplayName: "Resolution status", Description: "The parse-time resolution status: GO_SUM_ONLY, INCOMPLETE, or empty when unknown.", Type: "String"},
		{Name: "unresolvedImports", DisplayName: "Unresolved imports", Description: "Comma-separated import paths the toolchain could not map to a module (populated for INCOMPLETE).", Type: "String"},
		{Name: "resolutionError", DisplayName: "Resolution error", Description: "The toolchain/network failure reason when the build list could not be obtained (populated for GO_SUM_ONLY).", Type: "String"},
	},
)

// FindPartiallyResolvedGoMod flags each go.mod whose module graph did not fully
// resolve at parse time. GoModTidy, AddMissingGoModRequires, and
// RemoveUnusedGoModRequires all act only when the GoResolutionResult marker is
// RESOLVED with a package→module map; under any other status they safely no-op,
// leaving no signal that a needed add/remove was skipped rather than unnecessary.
//
// This recipe surfaces that condition: it marks the `module` directive with a
// search result and records a data-table row naming the module, its resolution
// status, the imports the toolchain could not map (INCOMPLETE), and the toolchain
// failure reason (GO_SUM_ONLY). It reports only and does not modify the go.mod.
type FindPartiallyResolvedGoMod struct {
	recipe.Base
}

func (r *FindPartiallyResolvedGoMod) Name() string {
	return "org.openrewrite.golang.migration.FindPartiallyResolvedGoMod"
}

func (r *FindPartiallyResolvedGoMod) DisplayName() string {
	return "Find go.mod files that could not be fully resolved offline"
}

func (r *FindPartiallyResolvedGoMod) Description() string {
	return "Find go.mod files whose module graph did not fully resolve at parse time, so `GoModTidy` skipped adding and removing requirements. " +
		"The offline no-op is safe but silent, making a module that badly needs tidying look identical to one already tidy. This recipe marks such go.mod files and records a data table naming the module, its resolution status, the unresolved imports, and the toolchain failure reason, so a run can be triaged and re-run once the modules resolve. It reports only and does not modify the go.mod."
}

func (r *FindPartiallyResolvedGoMod) Tags() []string { return []string{"gomod", "tidy", "search"} }

func (r *FindPartiallyResolvedGoMod) DataTables() []recipe.DataTableDescriptor {
	return []recipe.DataTableDescriptor{partiallyResolvedGoModTable.Descriptor()}
}

func (r *FindPartiallyResolvedGoMod) Editor() recipe.TreeVisitor {
	return preconditions.Check(
		preconditions.HasSourcePath("**/go.mod"),
		visitor.Init(&findPartiallyResolvedGoModVisitor{}),
	)
}

type findPartiallyResolvedGoModVisitor struct {
	visitor.GoVisitor
}

func (v *findPartiallyResolvedGoModVisitor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	mrr := java.FindMarker[golang.GoResolutionResult](gm.Markers)
	if mrr == nil || (mrr.ResolutionStatus == golang.GoResolutionResolved && len(mrr.PackageModules) > 0) {
		return gm
	}

	if ctx, ok := p.(*recipe.ExecutionContext); ok {
		partiallyResolvedGoModTable.InsertRow(ctx, PartiallyResolvedGoModRow{
			SourcePath:        gm.SourcePath,
			ModulePath:        mrr.ModulePath,
			ResolutionStatus:  string(mrr.ResolutionStatus),
			UnresolvedImports: strings.Join(mrr.UnresolvedImports, ", "),
			ResolutionError:   mrr.ResolutionError,
		})
	}

	return markModuleDirective(gm, unresolvedSearchResult(mrr.ResolutionStatus))
}

// unresolvedSearchResult is the search-result text for a go.mod that did not
// fully resolve, naming the status so a search view distinguishes the two ways
// resolution stops short.
func unresolvedSearchResult(status golang.GoResolutionStatus) string {
	if status == "" {
		return "go.mod not fully resolved offline; go mod tidy add/remove skipped"
	}
	return fmt.Sprintf("go.mod not fully resolved offline (%s); go mod tidy add/remove skipped", status)
}

// markModuleDirective returns gm with a search result attached to its `module`
// directive, or gm unchanged when it has no `module` directive.
func markModuleDirective(gm *golang.GoMod, text string) *golang.GoMod {
	for i, rp := range gm.Statements {
		d, ok := rp.Element.(*golang.GoModDirective)
		if !ok || d.Keyword != "module" {
			continue
		}
		rp.Element = d.WithMarkers(java.FoundSearchResult(d.Markers, text))
		statements := append([]java.RightPadded[golang.GoModStatement]{}, gm.Statements...)
		statements[i] = rp
		return gm.WithStatements(statements)
	}
	return gm
}
