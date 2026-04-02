/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
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

func (v *avoidReadAllInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *avoidReadAllInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *avoidReadAllInLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
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
		tree.MarkupWarn(mi.Markers, "ReadAll in loop; reads entire content into memory each iteration"),
	)
	return mi
}
