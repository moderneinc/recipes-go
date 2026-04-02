/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// FindMapRangeClear finds `for k := range m { delete(m, k) }` patterns where
// a map is cleared by ranging over it and deleting each key. In Go 1.21+,
// this is replaced with `clear(m)`.
type FindMapRangeClear struct {
	recipe.Base
}

func (r *FindMapRangeClear) Name() string {
	return "org.openrewrite.golang.codequality.FindMapRangeClear"
}
func (r *FindMapRangeClear) DisplayName() string { return "Replace map range-delete with clear()" }
func (r *FindMapRangeClear) Description() string {
	return "Replace `for k := range m { delete(m, k) }` with `clear(m)` (Go 1.21+)."
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

	// Build: clear(m) where m is forEach.Control.Iterable.
	// Strip the iterable's prefix (it had a space after the "range" keyword).
	mapExpr := stripExprPrefix(forEach.Control.Iterable)

	return &tree.MethodInvocation{
		Prefix: forEach.Prefix,
		Name:   &tree.Identifier{Name: "clear"},
		Arguments: tree.Container[tree.Expression]{
			Elements: []tree.RightPadded[tree.Expression]{
				{Element: mapExpr},
			},
		},
	}
}

// stripExprPrefix returns a copy of the expression with an empty prefix.
func stripExprPrefix(expr tree.Expression) tree.Expression {
	switch e := expr.(type) {
	case *tree.Identifier:
		return e.WithPrefix(tree.EmptySpace)
	case *tree.MethodInvocation:
		return e.WithPrefix(tree.EmptySpace)
	default:
		return expr
	}
}
