/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindDotImport finds dot imports (`import . "pkg"`). Dot imports pollute the
// local namespace and make it harder to understand where identifiers come from.
// golangci-lint: revive (dot-imports)
type FindDotImport struct {
	recipe.Base
}

func (r *FindDotImport) Name() string {
	return "org.openrewrite.golang.codequality.FindDotImport"
}
func (r *FindDotImport) DisplayName() string { return "Find dot imports" }
func (r *FindDotImport) Description() string {
	return "Find dot imports (`import . \"pkg\"`). Dot imports pollute the local namespace and make it harder to understand where identifiers come from."
}
func (r *FindDotImport) Tags() []string { return []string{"style", "lint"} }

func (r *FindDotImport) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "dot-imports", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *FindDotImport) Editor() recipe.TreeVisitor {
	return visitor.Init(&findDotImportVisitor{})
}

type findDotImportVisitor struct {
	visitor.GoVisitor
}

func (v *findDotImportVisitor) VisitImport(imp *tree.Import, p any) tree.J {
	imp = v.GoVisitor.VisitImport(imp, p).(*tree.Import)

	if imp.Alias == nil {
		return imp
	}

	aliasIdent := imp.Alias.Element
	if aliasIdent.Name != "." {
		return imp
	}

	imp = imp.WithMarkers(tree.FoundSearchResult(imp.Markers, "avoid dot import"))
	return imp
}
