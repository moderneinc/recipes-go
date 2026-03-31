/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// CheckErrorReturn finds multi-assignments where the last variable on the
// left-hand side is `_`, indicating a potentially discarded error return value.
// golangci-lint: errcheck
type CheckErrorReturn struct {
	recipe.Base
}

func (r *CheckErrorReturn) Name() string {
	return "org.openrewrite.golang.codequality.CheckErrorReturn"
}
func (r *CheckErrorReturn) DisplayName() string { return "Check error return value" }
func (r *CheckErrorReturn) Description() string {
	return "Find multi-assignments where the last return value is discarded with `_`, which may indicate an unchecked error."
}
func (r *CheckErrorReturn) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *CheckErrorReturn) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "errcheck", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *CheckErrorReturn) Editor() recipe.TreeVisitor {
	return visitor.Init(&checkErrorReturnVisitor{})
}

type checkErrorReturnVisitor struct {
	visitor.GoVisitor
}

func (v *checkErrorReturnVisitor) VisitMultiAssignment(ma *tree.MultiAssignment, p any) tree.J {
	ma = v.GoVisitor.VisitMultiAssignment(ma, p).(*tree.MultiAssignment)

	if len(ma.Variables) == 0 {
		return ma
	}

	// Check if the last LHS variable is the blank identifier `_`.
	lastVar := ma.Variables[len(ma.Variables)-1]
	ident, ok := lastVar.Element.(*tree.Identifier)
	if !ok || ident.Name != "_" {
		return ma
	}

	// Mark the blank identifier with a search result.
	marked := ident.WithMarkers(
		tree.FoundSearchResult(ident.Markers, "error return value discarded"),
	)
	vars := make([]tree.RightPadded[tree.Expression], len(ma.Variables))
	copy(vars, ma.Variables)
	vars[len(vars)-1] = tree.RightPadded[tree.Expression]{
		Element: marked,
		After:   lastVar.After,
		Markers: lastVar.Markers,
	}
	ma = ma.WithMarkers(ma.Markers) // shallow copy
	c := *ma
	c.Variables = vars
	return &c
}
