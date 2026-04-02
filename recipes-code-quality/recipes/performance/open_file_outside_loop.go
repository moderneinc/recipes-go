/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// OpenFileOutsideLoop finds `os.Open()` or `os.Create()` calls inside
// for/range loops. Opening files in tight loops should use a single open
// outside the loop.
type OpenFileOutsideLoop struct {
	recipe.Base
}

func (r *OpenFileOutsideLoop) Name() string {
	return "org.openrewrite.golang.codequality.OpenFileOutsideLoop"
}
func (r *OpenFileOutsideLoop) DisplayName() string { return "Open file outside loop" }
func (r *OpenFileOutsideLoop) Description() string {
	return "Find `os.Open()` or `os.Create()` calls inside for/range loops. Opening files in tight loops should use a single open outside the loop."
}
func (r *OpenFileOutsideLoop) Tags() []string { return []string{"performance"} }

func (r *OpenFileOutsideLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&openFileOutsideLoopVisitor{})
}

type openFileOutsideLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *openFileOutsideLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *openFileOutsideLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *openFileOutsideLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
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
		tree.MarkupInfo(mi.Markers, "file open in loop; consider opening file once outside loop"),
	)
	return mi
}
