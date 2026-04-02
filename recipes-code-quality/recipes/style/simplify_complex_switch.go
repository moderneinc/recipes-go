/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// SimplifyComplexSwitch finds switch statements with more than 10 cases.
// A switch with many cases suggests using a map or strategy pattern instead.
type SimplifyComplexSwitch struct {
	recipe.Base
}

func (r *SimplifyComplexSwitch) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyComplexSwitch"
}
func (r *SimplifyComplexSwitch) DisplayName() string { return "Simplify complex switch" }
func (r *SimplifyComplexSwitch) Description() string {
	return "Find switch statements with more than 10 cases. Consider using a map or strategy pattern."
}
func (r *SimplifyComplexSwitch) Tags() []string { return []string{"style"} }

func (r *SimplifyComplexSwitch) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifyComplexSwitchVisitor{})
}

type simplifyComplexSwitchVisitor struct {
	visitor.GoVisitor
}

func (v *simplifyComplexSwitchVisitor) VisitSwitch(sw *tree.Switch, p any) tree.J {
	sw = v.GoVisitor.VisitSwitch(sw, p).(*tree.Switch)

	if sw.Body == nil {
		return sw
	}

	count := 0
	for _, stmt := range sw.Body.Statements {
		if _, ok := stmt.Element.(*tree.Case); ok {
			count++
		}
	}

	if count <= 10 {
		return sw
	}

	sw = sw.WithMarkers(tree.MarkupInfo(sw.Markers, "switch has too many cases; consider using a map or strategy pattern"))
	return sw
}
