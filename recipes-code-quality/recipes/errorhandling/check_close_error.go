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
	// Preserve the leading prefix from the MethodInvocation on the blank identifier.
	prefix := closeLeadingPrefix(mi)

	blank := &java.Identifier{
		Prefix: prefix,
		Name:   "_",
	}

	// Move the leading whitespace from the MethodInvocation to the blank
	// identifier and give the MethodInvocation a single-space prefix so it
	// prints as `_ = f.Close()`.
	adjusted := adjustClosePrefix(mi)

	return &java.Assignment{
		Variable: blank,
		Value: java.LeftPadded[java.Expression]{
			Before:  java.SingleSpace,
			Element: adjusted,
		},
	}
}

// closeLeadingPrefix extracts the leading prefix from a MethodInvocation.
func closeLeadingPrefix(mi *java.MethodInvocation) java.Space {
	if mi.Select != nil {
		if ident, ok := mi.Select.Element.(*java.Identifier); ok {
			return ident.Prefix
		}
	}
	return mi.Prefix
}

// adjustClosePrefix returns a copy of the MethodInvocation with its
// leading prefix set to a single space (for the space after `=`).
func adjustClosePrefix(mi *java.MethodInvocation) *java.MethodInvocation {
	if mi.Select != nil {
		if ident, ok := mi.Select.Element.(*java.Identifier); ok {
			newSelect := *mi.Select
			newSelect.Element = ident.WithPrefix(java.SingleSpace)
			c := *mi
			c.Select = &newSelect
			return &c
		}
	}
	return mi
}
