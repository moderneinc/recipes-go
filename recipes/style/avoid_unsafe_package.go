/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidUnsafePackage finds any usage of the `unsafe` package. The unsafe package
// bypasses Go's type safety guarantees and should be avoided unless absolutely
// necessary. Common uses include unsafe.Pointer, unsafe.Sizeof, and
// unsafe.Alignof.
type AvoidUnsafePackage struct {
	recipe.Base
}

func (r *AvoidUnsafePackage) Name() string {
	return "org.openrewrite.golang.codequality.AvoidUnsafePackage"
}
func (r *AvoidUnsafePackage) DisplayName() string { return "Avoid unsafe package" }
func (r *AvoidUnsafePackage) Description() string {
	return "Find any usage of the `unsafe` package. The unsafe package bypasses Go's type safety guarantees and should be avoided unless absolutely necessary."
}
func (r *AvoidUnsafePackage) Tags() []string { return []string{"security"} }

func (r *AvoidUnsafePackage) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidUnsafePackageVisitor{})
}

type avoidUnsafePackageVisitor struct {
	visitor.GoVisitor
}

func (v *avoidUnsafePackageVisitor) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)

	ident, ok := fa.Target.(*java.Identifier)
	if !ok || ident.Name != "unsafe" {
		return fa
	}

	fa = fa.WithMarkers(java.MarkupWarn(fa.Markers, "unsafe package usage"))
	return fa
}

func (v *avoidUnsafePackageVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "unsafe" {
		return mi
	}

	mi = mi.WithMarkers(java.MarkupWarn(mi.Markers, "unsafe package usage"))
	return mi
}
