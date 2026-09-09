/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/google/uuid"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveSwitchTrueTag simplifies `switch true { ... }` statements by removing the
// redundant `true` tag. Go allows tagless switch (`switch { ... }`) which
// is equivalent and more idiomatic.
type RemoveSwitchTrueTag struct {
	recipe.Base
}

func (r *RemoveSwitchTrueTag) Name() string {
	return "org.openrewrite.golang.codequality.RemoveSwitchTrueTag"
}
func (r *RemoveSwitchTrueTag) DisplayName() string { return "Remove switch true tag" }
func (r *RemoveSwitchTrueTag) Description() string {
	return "Remove redundant `true` tag from `switch true { ... }` statements. Use a tagless switch instead."
}
func (r *RemoveSwitchTrueTag) Tags() []string { return []string{"simplification", "cleanup"} }

func (r *RemoveSwitchTrueTag) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifySwitchTrueVisitor{})
}

type simplifySwitchTrueVisitor struct {
	visitor.GoVisitor
}

func (v *simplifySwitchTrueVisitor) VisitSwitch(sw *java.Switch, p any) java.J {
	sw = v.GoVisitor.VisitSwitch(sw, p).(*java.Switch)

	if sw.Selector == nil {
		return sw
	}

	// The tag must be the identifier `true`.
	ident, ok := sw.Selector.Tree.Element.(*java.Identifier)
	if !ok || ident.Name != "true" {
		return sw
	}

	// Replace the selector with the empty one the parser emits for a tagless
	// `switch {}`: an Empty inner element and no leading space (the space before
	// `{` lives on the body's prefix).
	c := *sw
	c.Selector = &java.ControlParentheses{
		ID:      uuid.New(),
		Markers: java.Markers{ID: uuid.New()},
		Tree:    java.RightPadded[java.Expression]{Element: &java.Empty{ID: uuid.New(), Markers: java.Markers{ID: uuid.New()}}},
	}
	return &c
}
