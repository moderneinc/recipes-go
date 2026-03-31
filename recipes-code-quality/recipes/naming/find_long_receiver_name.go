/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindLongReceiverName finds method receivers with names longer than 2
// characters. Go convention is to use short, one- or two-letter receiver names.
type FindLongReceiverName struct {
	recipe.Base
}

func (r *FindLongReceiverName) Name() string {
	return "org.openrewrite.golang.codequality.FindLongReceiverName"
}
func (r *FindLongReceiverName) DisplayName() string { return "Find long receiver names" }
func (r *FindLongReceiverName) Description() string {
	return "Find method receivers with names longer than 2 characters. Go convention is to use short, one- or two-letter receiver names."
}
func (r *FindLongReceiverName) Tags() []string { return []string{"naming"} }

func (r *FindLongReceiverName) Editor() recipe.TreeVisitor {
	return visitor.Init(&findLongReceiverNameVisitor{})
}

type findLongReceiverNameVisitor struct {
	visitor.GoVisitor
}

func (v *findLongReceiverNameVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Receiver == nil || md.Name == nil {
		return md
	}

	// The receiver is a Container[Statement] with typically one parameter.
	for _, paramRP := range md.Receiver.Elements {
		vd, ok := paramRP.Element.(*tree.VariableDeclarations)
		if !ok {
			continue
		}

		for _, varRP := range vd.Variables {
			decl := varRP.Element
			if decl.Name == nil {
				continue
			}

			if len(decl.Name.Name) <= 2 {
				continue
			}

			// Mark the method name (the finding is about the method's receiver
			// naming convention).
			md = md.WithName(md.Name.WithMarkers(
				tree.FoundSearchResult(md.Name.Markers, "receiver name should be 1-2 characters"),
			))
			return md
		}
	}

	return md
}
