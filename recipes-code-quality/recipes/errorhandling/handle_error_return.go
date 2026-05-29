/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// HandleErrorReturn replaces the blank identifier `_` in the last position of
// a multi-assignment with `err`, capturing the previously discarded error value.
// `_, _ = f()` becomes `_, err = f()`.
// golangci-lint: errcheck
type HandleErrorReturn struct {
	recipe.Base
}

func (r *HandleErrorReturn) Name() string {
	return "org.openrewrite.golang.codequality.HandleErrorReturn"
}
func (r *HandleErrorReturn) DisplayName() string { return "Handle error return value" }
func (r *HandleErrorReturn) Description() string {
	return "Replace discarded `_` error return values with `err` to capture the error."
}
func (r *HandleErrorReturn) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *HandleErrorReturn) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "errcheck", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *HandleErrorReturn) Editor() recipe.TreeVisitor {
	return visitor.Init(&handleErrorReturnVisitor{})
}

type handleErrorReturnVisitor struct {
	visitor.GoVisitor
}

func (v *handleErrorReturnVisitor) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	ma = v.GoVisitor.VisitMultiAssignment(ma, p).(*golang.MultiAssignment)

	if len(ma.Variables) == 0 {
		return ma
	}

	// Check if the last LHS variable is the blank identifier `_`.
	lastVar := ma.Variables[len(ma.Variables)-1]
	ident, ok := lastVar.Element.(*java.Identifier)
	if !ok || ident.Name != "_" {
		return ma
	}

	// Replace `_` with `err` to capture the error value.
	replaced := ident.WithName("err")
	vars := make([]java.RightPadded[java.Expression], len(ma.Variables))
	copy(vars, ma.Variables)
	vars[len(vars)-1] = java.RightPadded[java.Expression]{
		Element: replaced,
		After:   lastVar.After,
		Markers: lastVar.Markers,
	}
	c := *ma
	c.Variables = vars
	return &c
}
