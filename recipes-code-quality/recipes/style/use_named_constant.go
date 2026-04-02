/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseNamedConstant finds numeric integer literals other than 0 and 1 that should
// be named constants. Magic numbers make code harder to understand and maintain.
// golangci-lint: mnd (magic number detector)
type UseNamedConstant struct {
	recipe.Base
}

func (r *UseNamedConstant) Name() string {
	return "org.openrewrite.golang.codequality.UseNamedConstant"
}
func (r *UseNamedConstant) DisplayName() string { return "Use named constants" }
func (r *UseNamedConstant) Description() string {
	return "Find numeric literals (other than 0 and 1) that should be named constants."
}
func (r *UseNamedConstant) Tags() []string { return []string{"style", "lint"} }

func (r *UseNamedConstant) Editor() recipe.TreeVisitor {
	return visitor.Init(&useNamedConstantVisitor{})
}

type useNamedConstantVisitor struct {
	visitor.GoVisitor
	insideConstOrVar bool
}

func (v *useNamedConstantVisitor) VisitVariableDeclarations(vd *tree.VariableDeclarations, p any) tree.J {
	// Skip literals inside const or var declarations.
	v.insideConstOrVar = true
	vd = v.GoVisitor.VisitVariableDeclarations(vd, p).(*tree.VariableDeclarations)
	v.insideConstOrVar = false
	return vd
}

func (v *useNamedConstantVisitor) VisitLiteral(lit *tree.Literal, p any) tree.J {
	lit = v.GoVisitor.VisitLiteral(lit, p).(*tree.Literal)

	if v.insideConstOrVar {
		return lit
	}

	if lit.Kind != tree.IntLiteral {
		return lit
	}

	// Allow common trivial values.
	if lit.Source == "0" || lit.Source == "1" {
		return lit
	}

	lit = lit.WithMarkers(
		tree.MarkupInfo(lit.Markers, "magic number; consider using a named constant"),
	)
	return lit
}
