/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindSwitchTrue simplifies `switch true { ... }` statements by removing the
// redundant `true` tag. Go allows tagless switch (`switch { ... }`) which
// is equivalent and more idiomatic.
type FindSwitchTrue struct {
	recipe.Base
}

func (r *FindSwitchTrue) Name() string {
	return "org.openrewrite.golang.codequality.FindSwitchTrue"
}
func (r *FindSwitchTrue) DisplayName() string { return "Simplify switch true" }
func (r *FindSwitchTrue) Description() string {
	return "Remove redundant `true` tag from `switch true { ... }` statements. Use a tagless switch instead."
}
func (r *FindSwitchTrue) Tags() []string { return []string{"simplification", "cleanup"} }

func (r *FindSwitchTrue) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifySwitchTrueVisitor{})
}

type simplifySwitchTrueVisitor struct {
	visitor.GoVisitor
}

func (v *simplifySwitchTrueVisitor) VisitSwitch(sw *tree.Switch, p any) tree.J {
	sw = v.GoVisitor.VisitSwitch(sw, p).(*tree.Switch)

	// Skip select statements
	if tree.HasMarker[tree.SelectStmt](sw.Markers) {
		return sw
	}

	// Must have a tag expression
	if sw.Tag == nil {
		return sw
	}

	// Tag must be the identifier `true`
	ident, ok := sw.Tag.Element.(*tree.Identifier)
	if !ok || ident.Name != "true" {
		return sw
	}

	// Remove the `true` tag. The space after the tag (before `{`) is in Tag.After,
	// which becomes unnecessary. The body block already has its own prefix.
	c := *sw
	c.Tag = nil
	return &c
}
