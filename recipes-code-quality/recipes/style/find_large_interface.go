/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindLargeInterface finds interfaces with more than 5 methods.
// Large interfaces violate the Interface Segregation Principle and should
// be split into smaller, more focused interfaces.
type FindLargeInterface struct {
	recipe.Base
}

func (r *FindLargeInterface) Name() string {
	return "org.openrewrite.golang.codequality.FindLargeInterface"
}
func (r *FindLargeInterface) DisplayName() string { return "Find large interfaces" }
func (r *FindLargeInterface) Description() string {
	return "Find interfaces with more than 5 methods. Large interfaces violate the Interface Segregation Principle."
}
func (r *FindLargeInterface) Tags() []string { return []string{"style", "lint"} }

func (r *FindLargeInterface) Editor() recipe.TreeVisitor {
	return visitor.Init(&findLargeInterfaceVisitor{})
}

type findLargeInterfaceVisitor struct {
	visitor.GoVisitor
}

func (v *findLargeInterfaceVisitor) VisitInterfaceType(it *tree.InterfaceType, p any) tree.J {
	it = v.GoVisitor.VisitInterfaceType(it, p).(*tree.InterfaceType)

	if it.Body == nil {
		return it
	}

	count := 0
	for _, s := range it.Body.Statements {
		if _, isEmpty := s.Element.(*tree.Empty); !isEmpty {
			count++
		}
	}

	if count <= 5 {
		return it
	}

	it = it.WithMarkers(tree.FoundSearchResult(it.Markers, "interface has too many methods"))
	return it
}
