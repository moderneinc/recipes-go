/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveRedundantRangeBlank simplifies `for i, _ := range s` by removing the
// blank identifier in the value position. Go allows `for i := range s`
// which is equivalent and more idiomatic.
type RemoveRedundantRangeBlank struct {
	recipe.Base
}

func (r *RemoveRedundantRangeBlank) Name() string {
	return "org.openrewrite.golang.codequality.RemoveRedundantRangeBlank"
}
func (r *RemoveRedundantRangeBlank) DisplayName() string { return "Remove redundant range blank" }
func (r *RemoveRedundantRangeBlank) Description() string {
	return "Remove the blank identifier from `for i, _ := range s` loops. Use `for i := range s` instead."
}
func (r *RemoveRedundantRangeBlank) Tags() []string { return []string{"simplification", "cleanup"} }

func (r *RemoveRedundantRangeBlank) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1005", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *RemoveRedundantRangeBlank) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifyRedundantRangeBlankVisitor{})
}

type simplifyRedundantRangeBlankVisitor struct {
	visitor.GoVisitor
}

func (v *simplifyRedundantRangeBlankVisitor) VisitForEachLoop(forEach *java.ForEachLoop, p any) java.J {
	forEach = v.GoVisitor.VisitForEachLoop(forEach, p).(*java.ForEachLoop)

	ctrl := forEach.Control

	// The loop targets and the `:=`/`=` operator live in a golang.MultiAssignment.
	ma, ok := ctrl.Variable.Element.(*golang.MultiAssignment)
	if !ok {
		return forEach
	}

	// Must have exactly two targets: `for k, v := range s`.
	if len(ma.Variables) != 2 {
		return forEach
	}

	// The value (second target) must be the blank identifier `_`.
	ident, ok := ma.Variables[1].Element.(*java.Identifier)
	if !ok || ident.Name != "_" {
		return forEach
	}

	// Drop the blank value, keeping just the key: `for k := range s`. The comma
	// and trailing space were carried on the key's After, which is no longer
	// printed once the key is the only (last) target.
	newMa := *ma
	key := ma.Variables[0]
	key.After = java.Space{}
	newMa.Variables = []java.RightPadded[java.Expression]{key}

	newCtrl := ctrl
	newCtrl.Variable.Element = &newMa

	c := *forEach
	c.Control = newCtrl
	return &c
}
