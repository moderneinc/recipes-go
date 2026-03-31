/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindJsonRawMessage finds usage of `json.RawMessage`. RawMessage defers
// JSON parsing and should be reviewed to ensure the deferred parsing is
// handled correctly.
type FindJsonRawMessage struct {
	recipe.Base
}

func (r *FindJsonRawMessage) Name() string {
	return "org.openrewrite.golang.codequality.FindJsonRawMessage"
}
func (r *FindJsonRawMessage) DisplayName() string { return "Find json.RawMessage usage" }
func (r *FindJsonRawMessage) Description() string {
	return "Find usage of `json.RawMessage`. RawMessage defers JSON parsing and should be reviewed to ensure deferred parsing is handled correctly."
}
func (r *FindJsonRawMessage) Tags() []string { return []string{"style"} }

func (r *FindJsonRawMessage) Editor() recipe.TreeVisitor {
	return visitor.Init(&findJsonRawMessageVisitor{})
}

type findJsonRawMessageVisitor struct {
	visitor.GoVisitor
}

func (v *findJsonRawMessageVisitor) VisitFieldAccess(fa *tree.FieldAccess, p any) tree.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*tree.FieldAccess)

	ident, ok := fa.Target.(*tree.Identifier)
	if !ok || ident.Name != "json" {
		return fa
	}

	if fa.Name.Element.Name != "RawMessage" {
		return fa
	}

	fa = fa.WithMarkers(tree.FoundSearchResult(fa.Markers, "json.RawMessage defers parsing; review for correctness"))
	return fa
}
