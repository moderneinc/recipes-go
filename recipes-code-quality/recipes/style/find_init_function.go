/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindInitFunction finds `func init()` declarations. Init functions make
// testing harder and have implicit ordering dependencies.
// golangci-lint: gochecknoinits
type FindInitFunction struct {
	recipe.Base
}

func (r *FindInitFunction) Name() string {
	return "org.openrewrite.golang.codequality.FindInitFunction"
}
func (r *FindInitFunction) DisplayName() string { return "Find init functions" }
func (r *FindInitFunction) Description() string {
	return "Find `func init()` declarations. Init functions make testing harder and have implicit ordering dependencies."
}
func (r *FindInitFunction) Tags() []string { return []string{"style", "lint"} }

func (r *FindInitFunction) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "gochecknoinits", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *FindInitFunction) Editor() recipe.TreeVisitor {
	return visitor.Init(&findInitFunctionVisitor{})
}

type findInitFunctionVisitor struct {
	visitor.GoVisitor
}

func (v *findInitFunctionVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil || md.Name.Name != "init" {
		return md
	}

	// Must be a free function (no receiver).
	if md.Receiver != nil {
		return md
	}

	// Must have no parameters (only the Empty sentinel).
	for _, param := range md.Parameters.Elements {
		if _, isEmpty := param.Element.(*tree.Empty); !isEmpty {
			return md
		}
	}

	// Must have no return type.
	if md.ReturnType != nil {
		return md
	}

	md = md.WithName(md.Name.WithMarkers(
		tree.FoundSearchResult(md.Name.Markers, "consider removing init function"),
	))
	return md
}
