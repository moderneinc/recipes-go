/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindAllNilReturn finds `return nil, nil` statements where all returned
// values are nil. Returning nil for both a value and an error might indicate
// a missing error or forgotten result.
type FindAllNilReturn struct {
	recipe.Base
}

func (r *FindAllNilReturn) Name() string {
	return "org.openrewrite.golang.codequality.FindAllNilReturn"
}
func (r *FindAllNilReturn) DisplayName() string { return "Find return with all nil values" }
func (r *FindAllNilReturn) Description() string {
	return "Find `return nil, nil` where all returned values are nil, which may indicate a missing error or result."
}
func (r *FindAllNilReturn) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *FindAllNilReturn) Editor() recipe.TreeVisitor {
	return visitor.Init(&findAllNilReturnVisitor{})
}

type findAllNilReturnVisitor struct {
	visitor.GoVisitor
}

func (v *findAllNilReturnVisitor) VisitReturn(ret *tree.Return, p any) tree.J {
	ret = v.GoVisitor.VisitReturn(ret, p).(*tree.Return)

	// Must have at least 2 return expressions.
	if len(ret.Expressions) < 2 {
		return ret
	}

	// All expressions must be the identifier "nil".
	for _, rp := range ret.Expressions {
		ident, ok := rp.Element.(*tree.Identifier)
		if !ok || ident.Name != "nil" {
			return ret
		}
	}

	ret = ret.WithMarkers(
		tree.FoundSearchResult(ret.Markers, "all return values are nil; possible missing error or result"),
	)
	return ret
}
