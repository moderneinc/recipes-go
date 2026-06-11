/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidReadAllInLoop finds `io.ReadAll()` or `ioutil.ReadAll()` calls inside
// for/range loops. These read entire content into memory on each iteration.
type AvoidReadAllInLoop struct {
	recipe.Base
}

func (r *AvoidReadAllInLoop) Name() string {
	return "org.openrewrite.golang.codequality.AvoidReadAllInLoop"
}
func (r *AvoidReadAllInLoop) DisplayName() string { return "Avoid ReadAll in loop" }
func (r *AvoidReadAllInLoop) Description() string {
	return "Find `io.ReadAll()` or `ioutil.ReadAll()` calls inside for/range loops. These read entire content into memory on each iteration."
}
func (r *AvoidReadAllInLoop) Tags() []string { return []string{"performance"} }

func (r *AvoidReadAllInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidReadAllInLoopVisitor{})
}

type avoidReadAllInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *avoidReadAllInLoopVisitor) VisitForLoop(forLoop *java.ForLoop, p any) java.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*java.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *avoidReadAllInLoopVisitor) VisitForEachLoop(forEach *java.ForEachLoop, p any) java.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*java.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *avoidReadAllInLoopVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok {
		return mi
	}

	if mi.Name.Name != "ReadAll" {
		return mi
	}

	if ident.Name != "io" && ident.Name != "ioutil" {
		return mi
	}

	mi = mi.WithMarkers(
		java.MarkupWarn(mi.Markers, "ReadAll in loop; reads entire content into memory each iteration"),
	)
	return mi
}
