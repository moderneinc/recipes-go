/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindFileOpenInLoop finds `os.Open()` or `os.Create()` calls inside
// for/range loops. Opening files in tight loops should use a single open
// outside the loop.
type FindFileOpenInLoop struct {
	recipe.Base
}

func (r *FindFileOpenInLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindFileOpenInLoop"
}
func (r *FindFileOpenInLoop) DisplayName() string { return "Find file open in loop" }
func (r *FindFileOpenInLoop) Description() string {
	return "Find `os.Open()` or `os.Create()` calls inside for/range loops. Opening files in tight loops should use a single open outside the loop."
}
func (r *FindFileOpenInLoop) Tags() []string { return []string{"performance"} }

func (r *FindFileOpenInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findFileOpenInLoopVisitor{})
}

type findFileOpenInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *findFileOpenInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *findFileOpenInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *findFileOpenInLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "os" {
		return mi
	}

	name := mi.Name.Name
	if name != "Open" && name != "Create" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "file open in loop; consider opening file once outside loop"),
	)
	return mi
}
