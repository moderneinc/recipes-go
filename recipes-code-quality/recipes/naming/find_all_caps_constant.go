/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"regexp"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindAllCapsConstant finds constant or variable names using ALL_CAPS with
// underscores. Go convention is MixedCaps, not ALL_CAPS_WITH_UNDERSCORES.
// golangci-lint: revive (var-naming)
type FindAllCapsConstant struct {
	recipe.Base
}

func (r *FindAllCapsConstant) Name() string {
	return "org.openrewrite.golang.codequality.FindAllCapsConstant"
}
func (r *FindAllCapsConstant) DisplayName() string { return "Find ALL_CAPS constant names" }
func (r *FindAllCapsConstant) Description() string {
	return "Find constant or variable names using ALL_CAPS_WITH_UNDERSCORES. Go convention is to use MixedCaps, not ALL_CAPS."
}
func (r *FindAllCapsConstant) Tags() []string { return []string{"naming"} }

func (r *FindAllCapsConstant) Editor() recipe.TreeVisitor {
	return visitor.Init(&findAllCapsConstantVisitor{})
}

// allCapsWithUnderscore matches names that are all uppercase letters/digits
// with at least one underscore (e.g. MAX_BUFFER_SIZE).
var allCapsWithUnderscore = regexp.MustCompile(`^[A-Z][A-Z0-9]*(_[A-Z0-9]+)+$`)

type findAllCapsConstantVisitor struct {
	visitor.GoVisitor
}

func (v *findAllCapsConstantVisitor) VisitVariableDeclarator(vd *tree.VariableDeclarator, p any) tree.J {
	vd = v.GoVisitor.VisitVariableDeclarator(vd, p).(*tree.VariableDeclarator)

	if vd.Name == nil {
		return vd
	}

	name := vd.Name.Name
	if !allCapsWithUnderscore.MatchString(name) {
		return vd
	}

	vd = vd.WithName(vd.Name.WithMarkers(
		tree.FoundSearchResult(vd.Name.Markers, "name uses ALL_CAPS_WITH_UNDERSCORES; Go convention is MixedCaps"),
	))
	return vd
}
