/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// KeepInterfacesSmall finds interfaces with more than 5 methods.
// Large interfaces violate the Interface Segregation Principle and should
// be split into smaller, more focused interfaces.
type KeepInterfacesSmall struct {
	recipe.Base
}

func (r *KeepInterfacesSmall) Name() string {
	return "org.openrewrite.golang.codequality.KeepInterfacesSmall"
}
func (r *KeepInterfacesSmall) DisplayName() string { return "Keep interfaces small" }
func (r *KeepInterfacesSmall) Description() string {
	return "Find interfaces with more than 5 methods. Large interfaces violate the Interface Segregation Principle."
}
func (r *KeepInterfacesSmall) Tags() []string { return []string{"style", "lint"} }

func (r *KeepInterfacesSmall) Editor() recipe.TreeVisitor {
	return visitor.Init(&keepInterfacesSmallVisitor{})
}

type keepInterfacesSmallVisitor struct {
	visitor.GoVisitor
}

func (v *keepInterfacesSmallVisitor) VisitInterfaceType(it *golang.InterfaceType, p any) java.J {
	it = v.GoVisitor.VisitInterfaceType(it, p).(*golang.InterfaceType)

	if it.Body == nil {
		return it
	}

	count := 0
	for _, s := range it.Body.Statements {
		if _, isEmpty := s.Element.(*java.Empty); !isEmpty {
			count++
		}
	}

	if count <= 5 {
		return it
	}

	it = it.WithMarkers(java.MarkupInfo(it.Markers, "interface has too many methods"))
	return it
}
