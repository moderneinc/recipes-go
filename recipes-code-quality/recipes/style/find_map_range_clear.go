/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindMapRangeClear finds `for k := range m { delete(m, k) }` patterns where
// a map is cleared by ranging over it and deleting each key. In Go 1.21+,
// use `clear(m)` instead.
type FindMapRangeClear struct {
	recipe.Base
}

func (r *FindMapRangeClear) Name() string {
	return "org.openrewrite.golang.codequality.FindMapRangeClear"
}
func (r *FindMapRangeClear) DisplayName() string { return "Find map range-delete pattern" }
func (r *FindMapRangeClear) Description() string {
	return "Find `for k := range m { delete(m, k) }` patterns. In Go 1.21+, use `clear(m)` instead."
}
func (r *FindMapRangeClear) Tags() []string { return []string{"style"} }

func (r *FindMapRangeClear) Editor() recipe.TreeVisitor {
	return visitor.Init(&findMapRangeClearVisitor{})
}

type findMapRangeClearVisitor struct {
	visitor.GoVisitor
}

func (v *findMapRangeClearVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)

	if forEach.Body == nil {
		return forEach
	}

	// The body must have exactly one real statement
	var onlyStmt tree.Statement
	count := 0
	for _, stmt := range forEach.Body.Statements {
		if _, isEmpty := stmt.Element.(*tree.Empty); !isEmpty {
			count++
			onlyStmt = stmt.Element
		}
	}
	if count != 1 || onlyStmt == nil {
		return forEach
	}

	// The single statement must be a call to delete(...)
	mi, ok := onlyStmt.(*tree.MethodInvocation)
	if !ok {
		return forEach
	}

	if mi.Select != nil || mi.Name.Name != "delete" {
		return forEach
	}

	forEach = forEach.WithMarkers(
		tree.FoundSearchResult(forEach.Markers, "range-delete pattern; consider using clear() in Go 1.21+"),
	)

	return forEach
}
