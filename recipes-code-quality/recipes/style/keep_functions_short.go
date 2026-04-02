/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// KeepFunctionsShort finds functions with more than 20 statements in their body.
// Long functions are harder to understand and maintain; consider splitting them
// into smaller, focused functions.
// golangci-lint: funlen
type KeepFunctionsShort struct {
	recipe.Base
}

func (r *KeepFunctionsShort) Name() string {
	return "org.openrewrite.golang.codequality.KeepFunctionsShort"
}
func (r *KeepFunctionsShort) DisplayName() string { return "Keep functions short" }
func (r *KeepFunctionsShort) Description() string {
	return "Find functions with more than 20 statements. Long functions are harder to understand and maintain."
}
func (r *KeepFunctionsShort) Tags() []string { return []string{"style", "lint"} }

func (r *KeepFunctionsShort) Editor() recipe.TreeVisitor {
	return visitor.Init(&keepFunctionsShortVisitor{})
}

type keepFunctionsShortVisitor struct {
	visitor.GoVisitor
}

func (v *keepFunctionsShortVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Body == nil || md.Name == nil {
		return md
	}

	count := countStatements(md.Body.Statements)
	if count <= 20 {
		return md
	}

	md = md.WithName(md.Name.WithMarkers(
		tree.MarkupInfo(md.Name.Markers, "function has too many statements"),
	))
	return md
}

// countStatements counts real statements, excluding Empty sentinels.
func countStatements(stmts []tree.RightPadded[tree.Statement]) int {
	count := 0
	for _, s := range stmts {
		if _, isEmpty := s.Element.(*tree.Empty); !isEmpty {
			count++
		}
	}
	return count
}
