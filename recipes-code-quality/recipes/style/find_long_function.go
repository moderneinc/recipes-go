/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindLongFunction finds functions with more than 20 statements in their body.
// Long functions are harder to understand and maintain; consider splitting them
// into smaller, focused functions.
// golangci-lint: funlen
type FindLongFunction struct {
	recipe.Base
}

func (r *FindLongFunction) Name() string {
	return "org.openrewrite.golang.codequality.FindLongFunction"
}
func (r *FindLongFunction) DisplayName() string { return "Find long functions" }
func (r *FindLongFunction) Description() string {
	return "Find functions with more than 20 statements. Long functions are harder to understand and maintain."
}
func (r *FindLongFunction) Tags() []string { return []string{"style", "lint"} }

func (r *FindLongFunction) Editor() recipe.TreeVisitor {
	return visitor.Init(&findLongFunctionVisitor{})
}

type findLongFunctionVisitor struct {
	visitor.GoVisitor
}

func (v *findLongFunctionVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Body == nil || md.Name == nil {
		return md
	}

	count := countStatements(md.Body.Statements)
	if count <= 20 {
		return md
	}

	md = md.WithName(md.Name.WithMarkers(
		tree.FoundSearchResult(md.Name.Markers, "function has too many statements"),
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
