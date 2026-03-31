/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindRandSeed finds calls to `rand.Seed()`. As of Go 1.20, the global
// random number generator is automatically seeded, making explicit calls
// to `rand.Seed` unnecessary and deprecated.
type FindRandSeed struct {
	recipe.Base
}

func (r *FindRandSeed) Name() string {
	return "org.openrewrite.golang.codequality.FindRandSeed"
}
func (r *FindRandSeed) DisplayName() string { return "Find rand.Seed usage" }
func (r *FindRandSeed) Description() string {
	return "Find calls to `rand.Seed()`. Deprecated since Go 1.20; the global random number generator is automatically seeded."
}
func (r *FindRandSeed) Tags() []string { return []string{"style", "deprecation"} }

func (r *FindRandSeed) Editor() recipe.TreeVisitor {
	return visitor.Init(&findRandSeedVisitor{})
}

type findRandSeedVisitor struct {
	visitor.GoVisitor
}

func (v *findRandSeedVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "rand" {
		return mi
	}

	if mi.Name.Name != "Seed" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "rand.Seed is deprecated since Go 1.20; automatic seeding is used"))
	return mi
}
