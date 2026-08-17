/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// CheckCloseError replaces bare `f.Close()` calls with `_ = f.Close()` to
// explicitly mark the discarded error. This satisfies the errcheck linter and
// makes the intent clear.
type CheckCloseError struct {
	recipe.Base
}

func (r *CheckCloseError) Name() string {
	return "org.openrewrite.golang.codequality.CheckCloseError"
}
func (r *CheckCloseError) DisplayName() string { return "Check Close() error" }
func (r *CheckCloseError) Description() string {
	return "Replace bare `f.Close()` with `_ = f.Close()` to explicitly mark the discarded error."
}
func (r *CheckCloseError) Tags() []string { return []string{"error-handling"} }

func (r *CheckCloseError) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "errcheck", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *CheckCloseError) Editor() recipe.TreeVisitor {
	return visitor.Init(&checkCloseErrorVisitor{})
}

type checkCloseErrorVisitor struct {
	visitor.GoVisitor
}

// Reports whether mi's method returns exactly one value, the only case where
// `_ = mi` compiles.
func returnsSingleValue(mi *java.MethodInvocation) bool {
	if mi.MethodType == nil || mi.MethodType.ReturnType == nil {
		return false
	}
	if _, isTuple := mi.MethodType.ReturnType.(*java.JavaTypeParameterized); isTuple {
		return false
	}
	return java.TypeSignature(mi.MethodType.ReturnType) != "void"
}

func (v *checkCloseErrorVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	// Match: x.Close() — any method named "Close" with a receiver.
	if mi.Select == nil || mi.Name.Name != "Close" {
		return mi
	}

	// Only wraps a Close() that stands alone as a statement; a call whose result
	// is consumed, such as `return x.Close()`, has a non-block parent.
	parent := v.Cursor().Parent()
	if parent == nil {
		return mi
	}
	if _, ok := parent.Value().(*java.Block); !ok {
		return mi
	}

	// `_ = x.Close()` only compiles when Close returns exactly one value; skip a
	// void or multi-value Close.
	if !returnsSingleValue(mi) {
		return mi
	}

	// Wrap: f.Close() → _ = f.Close()
	// The leading whitespace lives on the outermost element, so carry the
	// invocation's prefix onto the new assignment. The space after `=` lives on
	// the value's own (outermost) prefix.
	prefix := mi.GetPrefix()

	blank := &java.Identifier{
		Name: "_",
	}

	adjusted := *mi
	adjusted.Prefix = java.SingleSpace
	if mi.Select != nil {
		if ident, ok := mi.Select.Element.(*java.Identifier); ok {
			newSelect := *mi.Select
			newSelect.Element = ident.WithPrefix(java.EmptySpace)
			adjusted.Select = &newSelect
		}
	}

	return &java.Assignment{
		Prefix:   prefix,
		Variable: blank,
		Value: java.LeftPadded[java.Expression]{
			Before:  java.SingleSpace,
			Element: &adjusted,
		},
	}
}
