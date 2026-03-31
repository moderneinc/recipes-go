/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindTestMain finds `func TestMain(m *testing.M)` declarations. TestMain
// overrides the default test execution, which can affect all tests in the
// package. Flag these for awareness during code review.
type FindTestMain struct {
	recipe.Base
}

func (r *FindTestMain) Name() string {
	return "org.openrewrite.golang.codequality.FindTestMain"
}
func (r *FindTestMain) DisplayName() string { return "Find TestMain functions" }
func (r *FindTestMain) Description() string {
	return "Find `func TestMain(m *testing.M)` declarations. TestMain overrides the default test execution for the entire package."
}
func (r *FindTestMain) Tags() []string { return []string{"testing"} }

func (r *FindTestMain) Editor() recipe.TreeVisitor {
	return visitor.Init(&findTestMainVisitor{})
}

type findTestMainVisitor struct {
	visitor.GoVisitor
}

func (v *findTestMainVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil || md.Name.Name != "TestMain" {
		return md
	}

	// Must be a free function (no receiver).
	if md.Receiver != nil {
		return md
	}

	md = md.WithName(md.Name.WithMarkers(
		tree.FoundSearchResult(md.Name.Markers, "TestMain overrides default test execution"),
	))
	return md
}
