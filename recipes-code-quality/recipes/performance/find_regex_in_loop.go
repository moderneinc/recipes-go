/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindRegexInLoop finds calls to `regexp.Compile()` or `regexp.MustCompile()`
// inside for/range loops. Regex compilation is expensive and the compiled
// pattern should be reused; move the call outside the loop.
type FindRegexInLoop struct {
	recipe.Base
}

func (r *FindRegexInLoop) Name() string {
	return "org.openrewrite.golang.codequality.FindRegexInLoop"
}
func (r *FindRegexInLoop) DisplayName() string { return "Find regex compilation in loop" }
func (r *FindRegexInLoop) Description() string {
	return "Find `regexp.Compile()` or `regexp.MustCompile()` calls inside for/range loops. Compile the regex once outside the loop for better performance."
}
func (r *FindRegexInLoop) Tags() []string { return []string{"performance"} }

func (r *FindRegexInLoop) Editor() recipe.TreeVisitor {
	return visitor.Init(&findRegexInLoopVisitor{})
}

type findRegexInLoopVisitor struct {
	visitor.GoVisitor
	insideLoop int
}

func (v *findRegexInLoopVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *findRegexInLoopVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *findRegexInLoopVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "regexp" {
		return mi
	}

	if mi.Name.Name != "Compile" && mi.Name.Name != "MustCompile" {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "regex compilation in loop; compile once outside the loop"),
	)
	return mi
}
