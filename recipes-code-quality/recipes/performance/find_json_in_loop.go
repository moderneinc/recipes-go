/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindJsonInLoop finds calls to `json.Marshal()` or `json.Unmarshal()` inside
// for/range loops. JSON encoding/decoding in tight loops is expensive; consider
// using a pre-allocated encoder/decoder or restructuring to batch operations.
type FindJsonInLoop struct {
	recipe.Base
}

func (r *FindJsonInLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindJsonInLoop"
}
func (r *FindJsonInLoop) DisplayName() string { return "Find JSON marshal/unmarshal in loop" }
func (r *FindJsonInLoop) Description() string {
	return "Find `json.Marshal()` or `json.Unmarshal()` calls inside for/range loops. Consider using a pre-allocated encoder/decoder for better performance."
}
func (r *FindJsonInLoop) Tags() []string { return []string{"performance"} }

func (r *FindJsonInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findJsonInLoopVisitor{})
}

type findJsonInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *findJsonInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *findJsonInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *findJsonInLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "json" {
		return mi
	}

	if mi.Name.Name != "Marshal" && mi.Name.Name != "Unmarshal" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "json marshal/unmarshal in loop; consider using a pre-allocated encoder/decoder"),
	)
	return mi
}
