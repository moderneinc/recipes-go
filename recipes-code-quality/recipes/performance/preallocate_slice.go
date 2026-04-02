/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package performance

import (
	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// PreallocateSlice finds `append()` calls inside for/range loops where the
// slice could potentially be preallocated for better performance.
// golangci-lint: prealloc
type PreallocateSlice struct {
	recipe.Base
}

func (r *PreallocateSlice) Name() string {
	return "org.openrewrite.golang.codequality.PreallocateSlice"
}
func (r *PreallocateSlice) DisplayName() string { return "Preallocate slice" }
func (r *PreallocateSlice) Description() string {
	return "Find `append()` calls inside for/range loops where the slice could be preallocated."
}
func (r *PreallocateSlice) Tags() []string { return []string{"performance", "lint"} }

func (r *PreallocateSlice) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "prealloc", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *PreallocateSlice) Editor() recipe.TreeVisitor {
	return visitor.Init(&preallocateSliceVisitor{})
}

type preallocateSliceVisitor struct {
	visitor.GoVisitor
	insideLoop int // depth counter for nested loops
}

func (v *preallocateSliceVisitor) VisitForLoop(forLoop *tree.ForLoop, p any) tree.J {
	v.insideLoop++
	forLoop = v.GoVisitor.VisitForLoop(forLoop, p).(*tree.ForLoop)
	v.insideLoop--
	return forLoop
}

func (v *preallocateSliceVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	v.insideLoop++
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)
	v.insideLoop--
	return forEach
}

func (v *preallocateSliceVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if v.insideLoop == 0 {
		return mi
	}

	// Match: append(...) — built-in, so no Select and Name == "append".
	if mi.Select != nil || mi.Name.Name != "append" {
		return mi
	}

	// Mark the append call with a search result.
	mi = mi.WithMarkers(tree.MarkupInfo(mi.Markers, "consider preallocating slice"))
	return mi
}
