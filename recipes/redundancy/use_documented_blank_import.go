/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseDocumentedBlankImport finds blank imports (`import _ "pkg"`). Blank imports are
// used for side effects and should be documented with a comment explaining
// why the side-effect import is needed.
type UseDocumentedBlankImport struct {
	recipe.Base
}

func (r *UseDocumentedBlankImport) Name() string {
	return "org.openrewrite.golang.codequality.UseDocumentedBlankImport"
}
func (r *UseDocumentedBlankImport) DisplayName() string { return "Use documented blank imports" }
func (r *UseDocumentedBlankImport) Description() string {
	return "Find blank imports (`import _ \"pkg\"`). Blank imports are used for side effects and should be documented."
}
func (r *UseDocumentedBlankImport) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *UseDocumentedBlankImport) Editor() recipe.TreeVisitor {
	return visitor.Init(&useDocumentedBlankImportVisitor{})
}

type useDocumentedBlankImportVisitor struct {
	visitor.GoVisitor
}

func (v *useDocumentedBlankImportVisitor) VisitImport(imp *java.Import, p any) java.J {
	imp = v.GoVisitor.VisitImport(imp, p).(*java.Import)

	if imp.Alias == nil {
		return imp
	}

	aliasIdent := imp.Alias.Element
	if aliasIdent.Name != "_" {
		return imp
	}

	imp = imp.WithMarkers(
		java.MarkupInfo(imp.Markers, "blank import used for side effects"),
	)
	return imp
}
