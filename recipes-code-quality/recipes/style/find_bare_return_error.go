/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindBareReturnNilError finds `return nil, err` statements where the error
// is returned without wrapping it with additional context. Wrapping errors
// with `fmt.Errorf("context: %w", err)` makes debugging easier.
type FindBareReturnNilError struct {
	recipe.Base
}

func (r *FindBareReturnNilError) Name() string {
	return "org.openrewrite.golang.codequality.FindBareReturnNilError"
}
func (r *FindBareReturnNilError) DisplayName() string { return "Find bare return nil, err" }
func (r *FindBareReturnNilError) Description() string {
	return "Find `return nil, err` where the error is not wrapped with context. Consider using `fmt.Errorf(\"...: %w\", err)`."
}
func (r *FindBareReturnNilError) Tags() []string { return []string{"style", "errorhandling"} }

func (r *FindBareReturnNilError) Editor() recipe.TreeVisitor {
	return visitor.Init(&findBareReturnNilErrorVisitor{})
}

type findBareReturnNilErrorVisitor struct {
	visitor.GoVisitor
}

func (v *findBareReturnNilErrorVisitor) VisitReturn(ret *tree.Return, p any) tree.J {
	ret = v.GoVisitor.VisitReturn(ret, p).(*tree.Return)

	if len(ret.Expressions) < 2 {
		return ret
	}

	// First expression must be the nil identifier.
	firstIdent, firstOk := ret.Expressions[0].Element.(*tree.Identifier)
	if !firstOk || firstIdent.Name != "nil" {
		return ret
	}

	// Last expression must be the bare "err" identifier.
	lastIdent, lastOk := ret.Expressions[len(ret.Expressions)-1].Element.(*tree.Identifier)
	if !lastOk || lastIdent.Name != "err" {
		return ret
	}

	ret = ret.WithMarkers(
		tree.FoundSearchResult(ret.Markers, "return nil, err without wrapping error context"),
	)
	return ret
}
