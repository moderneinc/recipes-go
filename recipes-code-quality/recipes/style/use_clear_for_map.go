/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *findMapRangeClearVisitor) VisitForEachLoop(forEach *java.ForEachLoop, p any) java.J {
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*java.ForEachLoop)

	if forEach.Body == nil {
		return forEach
	}

	// The body must have exactly one real statement
	var onlyStmt java.Statement
	count := 0
	for _, stmt := range forEach.Body.Statements {
		if _, isEmpty := stmt.Element.(*java.Empty); !isEmpty {
			count++
			onlyStmt = stmt.Element
		}
	}
	if count != 1 || onlyStmt == nil {
		return forEach
	}

	// The single statement must be a call to delete(...)
	mi, ok := onlyStmt.(*java.MethodInvocation)
	if !ok {
		return forEach
	}

	if mi.Select != nil || mi.Name.Name != "delete" {
		return forEach
	}

	// Build: clear(m) where m is forEach.Control.Iterable.
	// Strip the iterable's prefix (it had a space after the "range" keyword).
	mapExpr := stripExprPrefix(forEach.Control.Iterable)

	return &java.MethodInvocation{
		Prefix: forEach.Prefix,
		Name:   &java.Identifier{Name: "clear"},
		Arguments: java.Container[java.Expression]{
			Elements: []java.RightPadded[java.Expression]{
				{Element: mapExpr},
			},
		},
	}
}

// stripExprPrefix returns a copy of the expression with an empty prefix.
func stripExprPrefix(expr java.Expression) java.Expression {
	switch e := expr.(type) {
	case *java.Identifier:
		return e.WithPrefix(java.EmptySpace)
	case *java.MethodInvocation:
		return e.WithPrefix(java.EmptySpace)
	default:
		return expr
	}
}
