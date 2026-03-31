/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindMagicNumber finds numeric integer literals other than 0 and 1 that should
// be named constants. Magic numbers make code harder to understand and maintain.
// golangci-lint: mnd (magic number detector)
type FindMagicNumber struct {
	recipe.Base
}

func (r *FindMagicNumber) Name() string {
	return "org.openrewrite.golang.codequality.FindMagicNumber"
}
func (r *FindMagicNumber) DisplayName() string { return "Find magic numbers" }
func (r *FindMagicNumber) Description() string {
	return "Find numeric literals (other than 0 and 1) that should be named constants."
}
func (r *FindMagicNumber) Tags() []string { return []string{"style", "lint"} }

func (r *FindMagicNumber) Editor() recipe.TreeVisitor {
	return visitor.Init(&findMagicNumberVisitor{})
}

type findMagicNumberVisitor struct {
	visitor.GoVisitor
	insideConstOrVar bool
}

func (v *findMagicNumberVisitor) VisitVariableDeclarations(vd *tree.VariableDeclarations, p any) tree.J {
	// Skip literals inside const or var declarations.
	v.insideConstOrVar = true
	vd = v.GoVisitor.VisitVariableDeclarations(vd, p).(*tree.VariableDeclarations)
	v.insideConstOrVar = false
	return vd
}

func (v *findMagicNumberVisitor) VisitLiteral(lit *tree.Literal, p any) tree.J {
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
		tree.FoundSearchResult(lit.Markers, "magic number; consider using a named constant"),
	)
	return lit
}
