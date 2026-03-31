/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindHttpCallInLoop finds `http.Get()` or `http.Post()` calls inside
// for/range loops. Making HTTP requests in tight loops can be slow.
type FindHttpCallInLoop struct {
	recipe.Base
}

func (r *FindHttpCallInLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindHttpCallInLoop"
}
func (r *FindHttpCallInLoop) DisplayName() string { return "Find HTTP call in loop" }
func (r *FindHttpCallInLoop) Description() string {
	return "Find `http.Get()` or `http.Post()` calls inside for/range loops. Making HTTP requests in tight loops can be slow."
}
func (r *FindHttpCallInLoop) Tags() []string { return []string{"performance"} }

func (r *FindHttpCallInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findHttpCallInLoopVisitor{})
}

type findHttpCallInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *findHttpCallInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *findHttpCallInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *findHttpCallInLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
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
		tree.FoundSearchResult(mi.Markers, "HTTP call in loop; making HTTP requests in tight loops can be slow"),
	)
	return mi
}
