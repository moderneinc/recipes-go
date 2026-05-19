/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AvoidInitFunction finds `func init()` declarations. Init functions make
// testing harder and have implicit ordering dependencies.
// golangci-lint: gochecknoinits
type AvoidInitFunction struct {
	recipe.Base
}

func (r *AvoidInitFunction) Name() string {
	return "org.openrewrite.golang.codequality.AvoidInitFunction"
}
func (r *AvoidInitFunction) DisplayName() string { return "Avoid init functions" }
func (r *AvoidInitFunction) Description() string {
	return "Find `func init()` declarations. Init functions make testing harder and have implicit ordering dependencies."
}
func (r *AvoidInitFunction) Tags() []string { return []string{"style", "lint"} }

func (r *AvoidInitFunction) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "gochecknoinits", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *AvoidInitFunction) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidInitFunctionVisitor{})
}

type avoidInitFunctionVisitor struct {
	visitor.GoVisitor
}

func (v *avoidInitFunctionVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
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
		tree.MarkupInfo(md.Name.Markers, "consider removing init function"),
	))
	return md
}
