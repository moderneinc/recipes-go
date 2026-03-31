/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindTestFatal finds `t.Fatal()` and `t.Fatalf()` calls. These methods
// abort the test immediately and, when called from a goroutine other than
// the test function's goroutine (Go 1.16+), cause a panic. Consider using
// `t.Error`/`t.Errorf` instead, especially inside goroutines.
type FindTestFatal struct {
	recipe.Base
}

func (r *FindTestFatal) Name() string {
	return "org.openrewrite.golang.codequality.FindTestFatal"
}
func (r *FindTestFatal) DisplayName() string { return "Find t.Fatal calls" }
func (r *FindTestFatal) Description() string {
	return "Find `t.Fatal()` and `t.Fatalf()` calls. These abort the test immediately and panic when called from a goroutine other than the test function's goroutine."
}
func (r *FindTestFatal) Tags() []string { return []string{"testing"} }

func (r *FindTestFatal) Editor() recipe.TreeVisitor {
	return visitor.Init(&findTestFatalVisitor{})
}

type findTestFatalVisitor struct {
	visitor.GoVisitor
}

func (v *findTestFatalVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "t" {
		return mi
	}

	if !strings.HasPrefix(mi.Name.Name, "Fatal") {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "t.Fatal call found; consider t.Error in goroutines"),
	)
	return mi
}
