/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// BatchHttpCalls finds `http.Get()` or `http.Post()` calls inside
// for/range loops. Making HTTP requests in tight loops can be slow.
type BatchHttpCalls struct {
	recipe.Base
}

func (r *BatchHttpCalls) Name() string {
	return "org.openrewrite.golang.codequality.BatchHttpCalls"
}
func (r *BatchHttpCalls) DisplayName() string { return "Batch HTTP calls" }
func (r *BatchHttpCalls) Description() string {
	return "Find `http.Get()` or `http.Post()` calls inside for/range loops. Making HTTP requests in tight loops can be slow."
}
func (r *BatchHttpCalls) Tags() []string { return []string{"performance"} }

func (r *BatchHttpCalls) Editor() recipe.TreeVisitor {
	return visitor.Init(&batchHttpCallsVisitor{})
}

type batchHttpCallsVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *batchHttpCallsVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *batchHttpCallsVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *batchHttpCallsVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "http" {
		return mi
	}

	name := mi.Name.Name
	if name != "Get" && name != "Post" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.MarkupWarn(mi.Markers, "HTTP call in loop; making HTTP requests in tight loops can be slow"),
	)
	return mi
}
