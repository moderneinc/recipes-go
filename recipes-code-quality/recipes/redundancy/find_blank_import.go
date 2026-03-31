/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindBlankImport finds blank imports (`import _ "pkg"`). Blank imports are
// used for side effects and should be documented with a comment explaining
// why the side-effect import is needed.
type FindBlankImport struct {
	recipe.Base
}

func (r *FindBlankImport) Name() string {
	return "org.openrewrite.golang.codequality.FindBlankImport"
}
func (r *FindBlankImport) DisplayName() string { return "Find blank imports" }
func (r *FindBlankImport) Description() string {
	return "Find blank imports (`import _ \"pkg\"`). Blank imports are used for side effects and should be documented."
}
func (r *FindBlankImport) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *FindBlankImport) Editor() recipe.TreeVisitor {
	return visitor.Init(&findBlankImportVisitor{})
}

type findBlankImportVisitor struct {
	visitor.GoVisitor
}

func (v *findBlankImportVisitor) VisitImport(imp *tree.Import, p any) tree.J {
	imp = v.GoVisitor.VisitImport(imp, p).(*tree.Import)

	if imp.Alias == nil {
		return imp
	}

	aliasIdent := imp.Alias.Element
	if aliasIdent.Name != "_" {
		return imp
	}

	imp = imp.WithMarkers(
		tree.FoundSearchResult(imp.Markers, "blank import used for side effects"),
	)
	return imp
}
