/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseMeaningfulReturnValues finds `return nil, nil` statements where all returned
// values are nil. Returning nil for both a value and an error might indicate
// a missing error or forgotten result.
type UseMeaningfulReturnValues struct {
	recipe.Base
}

func (r *UseMeaningfulReturnValues) Name() string {
	return "org.openrewrite.golang.codequality.UseMeaningfulReturnValues"
}
func (r *UseMeaningfulReturnValues) DisplayName() string {
	return "Use meaningful return values"
}
func (r *UseMeaningfulReturnValues) Description() string {
	return "Find `return nil, nil` where all returned values are nil, which may indicate a missing error or result."
}
func (r *UseMeaningfulReturnValues) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *UseMeaningfulReturnValues) Editor() recipe.TreeVisitor {
	return visitor.Init(&useMeaningfulReturnValuesVisitor{})
}

type useMeaningfulReturnValuesVisitor struct {
	visitor.GoVisitor
}

func (v *useMeaningfulReturnValuesVisitor) VisitReturn(ret *tree.Return, p any) tree.J {
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
		tree.MarkupInfo(ret.Markers, "all return values are nil; possible missing error or result"),
	)
	return ret
}
