/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindRedundantRangeBlank simplifies `for i, _ := range s` by removing the
// blank identifier in the value position. Go allows `for i := range s`
// which is equivalent and more idiomatic.
type FindRedundantRangeBlank struct {
	recipe.Base
}

func (r *FindRedundantRangeBlank) Name() string {
	return "org.openrewrite.golang.codequality.FindRedundantRangeBlank"
}
func (r *FindRedundantRangeBlank) DisplayName() string { return "Remove redundant range blank" }
func (r *FindRedundantRangeBlank) Description() string {
	return "Remove the blank identifier from `for i, _ := range s` loops. Use `for i := range s` instead."
}
func (r *FindRedundantRangeBlank) Tags() []string { return []string{"simplification", "cleanup"} }

func (r *FindRedundantRangeBlank) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifyRedundantRangeBlankVisitor{})
}

type simplifyRedundantRangeBlankVisitor struct {
	visitor.GoVisitor
}

func (v *simplifyRedundantRangeBlankVisitor) VisitForEachLoop(forEach *tree.ForEachLoop, p any) tree.J {
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*tree.ForEachLoop)

	ctrl := forEach.Control

	// Must have a value in the second position
	if ctrl.Value == nil {
		return forEach
	}

	// Value must be the blank identifier `_`
	ident, ok := ctrl.Value.Element.(*tree.Identifier)
	if !ok || ident.Name != "_" {
		return forEach
	}

	// Remove the blank value. The comma and space before `_` are in Key.After,
	// so we also need to trim the Key's trailing space to remove `, _`.
	newCtrl := ctrl
	newCtrl.Value = nil
	// Reset Key.After to a single space (removing the `, _` trailing formatting)
	if newCtrl.Key != nil {
		k := *newCtrl.Key
		k.After = tree.Space{Whitespace: " "}
		newCtrl.Key = &k
	}

	c := *forEach
	c.Control = newCtrl
	return &c
}
