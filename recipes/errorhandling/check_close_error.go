/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
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

func (r *CheckCloseError) Editor() recipe.TreeVisitor {
	return visitor.Init(&checkCloseErrorVisitor{})
}

type checkCloseErrorVisitor struct {
	visitor.GoVisitor
	insideAssignment int
}

func (v *checkCloseErrorVisitor) VisitAssignment(assign *java.Assignment, p any) java.J {
	v.insideAssignment++
	assign = v.GoVisitor.VisitAssignment(assign, p).(*java.Assignment)
	v.insideAssignment--
	return assign
}

func (v *checkCloseErrorVisitor) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	v.insideAssignment++
	ma = v.GoVisitor.VisitMultiAssignment(ma, p).(*golang.MultiAssignment)
	v.insideAssignment--
	return ma
}

func (v *checkCloseErrorVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	// Match: x.Close() — any method named "Close" with a receiver.
	if mi.Select == nil || mi.Name.Name != "Close" {
		return mi
	}

	// Only transform bare statement calls. If this MethodInvocation is already
	// the RHS of an assignment, skip it.
	if v.insideAssignment > 0 {
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
