/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// WrapErrorWithContext finds bare `return err` statements that should wrap
// the error with context via `fmt.Errorf("context: %w", err)`.
type WrapErrorWithContext struct {
	recipe.Base
}

func (r *WrapErrorWithContext) Name() string {
	return "org.openrewrite.golang.codequality.WrapErrorWithContext"
}
func (r *WrapErrorWithContext) DisplayName() string { return "Wrap error with context" }
func (r *WrapErrorWithContext) Description() string {
	return "Find bare `return err` statements that should wrap the error with additional context."
}
func (r *WrapErrorWithContext) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *WrapErrorWithContext) Editor() recipe.TreeVisitor {
	return visitor.Init(&wrapErrorWithContextVisitor{})
}

type wrapErrorWithContextVisitor struct {
	visitor.GoVisitor
}

func (v *wrapErrorWithContextVisitor) VisitReturn(ret *tree.Return, p any) tree.J {
	ret = v.GoVisitor.VisitReturn(ret, p).(*tree.Return)

	// Match: return with a single expression that is an identifier named "err".
	if len(ret.Expressions) != 1 {
		return ret
	}

	expr := ret.Expressions[0].Element
	ident, ok := expr.(*tree.Identifier)
	if !ok || ident.Name != "err" {
		return ret
	}

	ret = ret.WithMarkers(
		tree.FoundSearchResult(ret.Markers, "bare return err; consider wrapping with fmt.Errorf"),
	)
	return ret
}
