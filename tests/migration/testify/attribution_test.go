/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify_test

import (
	"strings"
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/testify"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

func TestRequireNoErrorCarriesItsTypes(t *testing.T) {
	tree := rewritten(t, &testify.AdoptTestifyRequireNoError{}, `
		package sample

		import "testing"

		func do() error { return nil }

		func TestThing(t *testing.T) {
			err := do()
			if err != nil {
				t.Fatalf("do: %v", err)
			}
		}
	`)

	test.ExpectMethodType(t, tree, "NoError", "github.com/stretchr/testify/require")
	call := findCall(tree, "NoError")
	if call == nil {
		t.Fatal("no require.NoError call in the rewritten tree")
	}
	qualifier, ok := call.Select.Element.(*java.Identifier)
	if !ok || qualifier.Type == nil {
		t.Fatalf("qualifier %v has no type", call.Select.Element)
	}
	recv, ok := call.Arguments.Elements[0].Element.(*java.Identifier)
	if !ok || recv.Type == nil {
		t.Fatalf("receiver argument %v has no type", call.Arguments.Elements[0].Element)
	}
}

// rewritten applies a recipe outside RewriteRun, which compares printed source
// and so cannot see attribution.
func rewritten(t *testing.T, r recipe.Recipe, src string) java.Tree {
	t.Helper()
	cu, err := parser.NewGoParser().Parse("main_test.go", test.TrimIndent(strings.TrimPrefix(src, "\n")))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ctx := recipe.NewExecutionContext()
	editor := r.Editor()
	var tree java.Tree = cu
	if out := editor.Visit(tree, ctx); out != nil {
		tree = out
	}
	return visitor.DrainAfterVisits(editor, tree, ctx)
}

func findCall(root java.Tree, name string) *java.MethodInvocation {
	c := visitor.Init(&callFinder{name: name})
	c.Visit(root, nil)
	return c.found
}

type callFinder struct {
	visitor.GoVisitor
	name  string
	found *java.MethodInvocation
}

func (v *callFinder) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if v.found == nil && mi.Name != nil && mi.Name.Name == v.name {
		v.found = mi
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}
