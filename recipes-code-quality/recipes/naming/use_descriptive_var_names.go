/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseDescriptiveVarNames finds variable declarations with single-letter names that
// are not conventional short names. Go convention allows short names like i, j,
// k for loops and n, x, y for common uses, but other single-letter names hurt
// readability.
type UseDescriptiveVarNames struct {
	recipe.Base
}

func (r *UseDescriptiveVarNames) Name() string {
	return "org.openrewrite.golang.codequality.UseDescriptiveVarNames"
}
func (r *UseDescriptiveVarNames) DisplayName() string { return "Use descriptive variable names" }
func (r *UseDescriptiveVarNames) Description() string {
	return "Find variable declarations with single-letter names that are not conventional short names (i, j, k, n, x, y, r, w, t, s, b, v)."
}
func (r *UseDescriptiveVarNames) Tags() []string { return []string{"naming"} }

func (r *UseDescriptiveVarNames) Editor() recipe.TreeVisitor {
	return visitor.Init(&useDescriptiveVarNamesVisitor{})
}

// conventionalShortNames is the set of single-letter variable names that are
// conventional in Go (loop indices, common abbreviations, etc.).
var conventionalShortNames = map[string]bool{
	"i": true, "j": true, "k": true,
	"n": true, "x": true, "y": true,
	"r": true, "w": true, "t": true,
	"s": true, "b": true, "v": true,
}

type useDescriptiveVarNamesVisitor struct {
	visitor.GoVisitor
}

func (v *useDescriptiveVarNamesVisitor) VisitVariableDeclarator(vd *tree.VariableDeclarator, p any) tree.J {
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
		tree.MarkupInfo(vd.Name.Markers, "single-letter variable name is not a conventional short name"),
	))
	return vd
}
