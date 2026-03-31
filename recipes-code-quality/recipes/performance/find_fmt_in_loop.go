/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindFmtInLoop finds calls to `fmt.Sprintf`, `fmt.Sprint`, or `fmt.Fprintf`
// inside for/range loops. These functions allocate on every call; in hot loops,
// prefer direct string operations or strconv for better performance.
type FindFmtInLoop struct {
	recipe.Base
}

func (r *FindFmtInLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindFmtInLoop"
}
func (r *FindFmtInLoop) DisplayName() string { return "Find fmt formatting in loop" }
func (r *FindFmtInLoop) Description() string {
	return "Find `fmt.Sprintf`, `fmt.Sprint`, or `fmt.Fprintf` calls inside for/range loops. These allocate on every call; prefer direct string operations or strconv."
}
func (r *FindFmtInLoop) Tags() []string { return []string{"performance"} }

func (r *FindFmtInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findFmtInLoopVisitor{})
}

type findFmtInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *findFmtInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *findFmtInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *findFmtInLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "fmt" {
		return mi
	}

	name := mi.Name.Name
	if name != "Sprintf" && name != "Sprint" && name != "Fprintf" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "fmt formatting in loop; allocates on every call, prefer strconv or direct string operations"),
	)
	return mi
}
