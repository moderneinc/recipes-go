/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"strings"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindGetterPrefix finds methods with a "Get" prefix. In Go, getters should not
// have the "Get" prefix — `GetName()` should be `Name()`.
// golangci-lint: revive (exported)
type FindGetterPrefix struct {
	recipe.Base
}

func (r *FindGetterPrefix) Name() string {
	return "org.openrewrite.golang.codequality.FindGetterPrefix"
}
func (r *FindGetterPrefix) DisplayName() string { return "Find getter methods with Get prefix" }
func (r *FindGetterPrefix) Description() string {
	return "Find methods with a \"Get\" prefix. Go convention is that getters should not have the \"Get\" prefix."
}
func (r *FindGetterPrefix) Tags() []string { return []string{"naming"} }

func (r *FindGetterPrefix) Editor() recipe.TreeVisitor {
	return visitor.Init(&findGetterPrefixVisitor{})
}

type findGetterPrefixVisitor struct {
	visitor.GoVisitor
}

func (v *findGetterPrefixVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*tree.MethodDeclaration)

	if md.Name == nil {
		return md
	}

	// Only check methods (with a receiver), not free functions.
	if md.Receiver == nil {
		return md
	}

	funcName := md.Name.Name
	if len(funcName) <= 3 {
		return md
	}

	if !strings.HasPrefix(funcName, "Get") {
		return md
	}

	md = md.WithName(md.Name.WithMarkers(
		tree.FoundSearchResult(md.Name.Markers, "getter should not have Get prefix"),
	))
	return md
}
