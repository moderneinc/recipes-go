/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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
	insideArrayType  bool
}

func (v *useNamedConstantVisitor) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	// Skip literals inside const or var declarations.
	v.insideConstOrVar = true
	vd = v.GoVisitor.VisitVariableDeclarations(vd, p).(*java.VariableDeclarations)
	v.insideConstOrVar = false
	return vd
}

func (v *useNamedConstantVisitor) VisitGoArrayType(at *golang.ArrayType, p any) java.J {
	// Skip the declared array length, e.g. the `3` in `[3]int`; it is part of the
	// type, not a magic number.
	v.insideArrayType = true
	at = v.GoVisitor.VisitGoArrayType(at, p).(*golang.ArrayType)
	v.insideArrayType = false
	return at
}

func (v *useNamedConstantVisitor) VisitLiteral(lit *java.Literal, p any) java.J {
	lit = v.GoVisitor.VisitLiteral(lit, p).(*java.Literal)

	if v.insideConstOrVar || v.insideArrayType {
		return lit
	}

	if !matcher.IsInt(lit.Type) {
		return lit
	}

	// Allow common trivial values.
	if lit.Source == "0" || lit.Source == "1" {
		return lit
	}

	lit = lit.WithMarkers(
		java.MarkupInfo(lit.Markers, "magic number; consider using a named constant"),
	)
	return lit
}
