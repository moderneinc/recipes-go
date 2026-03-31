/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindSingleLetterVar finds variable declarations with single-letter names that
// are not conventional short names. Go convention allows short names like i, j,
// k for loops and n, x, y for common uses, but other single-letter names hurt
// readability.
type FindSingleLetterVar struct {
	recipe.Base
}

func (r *FindSingleLetterVar) Name() string {
	return "org.openrewrite.golang.codequality.FindSingleLetterVar"
}
func (r *FindSingleLetterVar) DisplayName() string { return "Find single-letter variable names" }
func (r *FindSingleLetterVar) Description() string {
	return "Find variable declarations with single-letter names that are not conventional short names (i, j, k, n, x, y, r, w, t, s, b, v)."
}
func (r *FindSingleLetterVar) Tags() []string { return []string{"naming"} }

func (r *FindSingleLetterVar) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSingleLetterVarVisitor{})
}

// conventionalShortNames is the set of single-letter (and short) variable names
// considered idiomatic in Go.
var conventionalShortNames = map[string]bool{
	"i": true, "j": true, "k": true,
	"n": true, "x": true, "y": true,
	"r": true, "w": true, "t": true,
	"s": true, "b": true, "v": true,
	"_": true,
}

type findSingleLetterVarVisitor struct {
	visitor.GoVisitor
}

func (v *findSingleLetterVarVisitor) VisitVariableDeclarator(vd *tree.VariableDeclarator, p any) tree.J {
	vd = v.GoVisitor.VisitVariableDeclarator(vd, p).(*tree.VariableDeclarator)

	if vd.Name == nil {
		return vd
	}

	name := vd.Name.Name
	if len(name) != 1 {
		return vd
	}

	if conventionalShortNames[name] {
		return vd
	}

	vd = vd.WithName(vd.Name.WithMarkers(
		tree.FoundSearchResult(vd.Name.Markers, "single-letter variable name is not a conventional short name"),
	))
	return vd
}
