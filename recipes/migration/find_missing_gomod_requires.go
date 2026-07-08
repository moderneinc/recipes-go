/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// FindMissingGoModRequires flags imports of third-party packages that no
// `require` (or `replace`) directive in the module's go.mod covers. These are
// exactly the requirements `go mod tidy` would add; the recipe cannot add them
// itself because resolving the version requires network access.
type FindMissingGoModRequires struct {
	recipe.Base
}

func (r *FindMissingGoModRequires) Name() string {
	return "org.openrewrite.golang.migration.FindMissingGoModRequires"
}

func (r *FindMissingGoModRequires) DisplayName() string {
	return "Find missing go.mod requirements"
}

func (r *FindMissingGoModRequires) Description() string {
	return "Find imports of third-party packages that are not covered by any `require` directive in the module's go.mod. " +
		"These are the requirements `go mod tidy` would add; adding them automatically is not possible offline because it requires resolving module versions over the network."
}

func (r *FindMissingGoModRequires) Tags() []string { return []string{"gomod", "tidy", "search"} }

func (r *FindMissingGoModRequires) Editor() recipe.TreeVisitor {
	return visitor.Init(&findMissingGoModRequiresVisitor{})
}

type findMissingGoModRequiresVisitor struct {
	visitor.GoVisitor
}

func (v *findMissingGoModRequiresVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)

	mrr := java.FindMarker[golang.GoResolutionResult](cu.Markers)
	if mrr == nil || cu.Imports == nil {
		return cu
	}

	elements := cu.Imports.Elements
	newElements := make([]java.RightPadded[*java.Import], len(elements))
	changed := false
	for i, rp := range elements {
		if isMissingRequire(importPathOf(rp.Element), mrr) {
			rp.Element = rp.Element.WithMarkers(
				java.FoundSearchResult(rp.Element.Markers, "missing go.mod requirement"),
			)
			changed = true
		}
		newElements[i] = rp
	}
	if !changed {
		return cu
	}
	imports := *cu.Imports
	imports.Elements = newElements
	cu = cu.WithImports(&imports)
	return cu
}

func isMissingRequire(importPath string, mrr *golang.GoResolutionResult) bool {
	if importPath == "" || isStdlibImport(importPath) {
		return false
	}
	if mrr.ModulePath != "" && moduleProvides(mrr.ModulePath, importPath) {
		return false
	}
	for i := range mrr.Requires {
		if moduleProvides(mrr.Requires[i].ModulePath, importPath) {
			return false
		}
	}
	for i := range mrr.Replaces {
		if moduleProvides(mrr.Replaces[i].OldPath, importPath) {
			return false
		}
	}
	return true
}
