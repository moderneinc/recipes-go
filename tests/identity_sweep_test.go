/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package tests

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Flags a recipe that rebuilds the tree while printing identically: the harness
// only catches this where a no-change test happens to exist.
func TestNoChangeMeansSamePointer(t *testing.T) {
	byType := map[string][]recipe.Recipe{}
	for _, r := range registeredRecipes() {
		byType[reflect.TypeOf(r).Elem().Name()] = append(byType[reflect.TypeOf(r).Elem().Name()], r)
	}
	offenders := map[string]map[string]bool{}
	for _, sn := range recipeSnippets(t) {
		src := test.TrimIndent(strings.TrimPrefix(sn.src, "\n"))
		cu, err := parser.NewGoParser().Parse("main.go", src)
		if err != nil {
			continue
		}
		before := java.J(cu)
		for _, typeName := range sn.recipes {
			for _, r := range byType[typeName] {
				after, err := runToCompletion(r, cu)
				if err != nil || after == nil {
					continue
				}
				aj, ok := after.(java.J)
				if !ok {
					continue
				}
				if aj == before {
					continue // correctly unchanged
				}
				if printer.Print(aj) == src && markerCount(aj) == markerCount(before) {
					if offenders[typeName] == nil {
						offenders[typeName] = map[string]bool{}
					}
					offenders[typeName][sn.file] = true
				}
			}
		}
	}
	if len(offenders) == 0 {
		t.Log("no recipe rebuilds a tree it did not change")
		return
	}
	names := make([]string, 0, len(offenders))
	for n := range offenders {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		files := make([]string, 0)
		for f := range offenders[n] {
			files = append(files, f)
		}
		sort.Strings(files)
		t.Errorf("%s rebuilt an unchanged tree in %v", n, files)
	}
}

// markerCount totals every marker in a tree: a find recipe reports by attaching
// one, which the printed source cannot show.
func markerCount(tree java.Tree) int {
	c := visitor.Init(&markerCounter{})
	c.Visit(tree, nil)
	return c.n
}

type markerCounter struct {
	visitor.GoVisitor
	n int
}

func (v *markerCounter) Visit(t java.Tree, p any) java.Tree {
	if j, ok := t.(java.J); ok && j != nil {
		v.n += len(j.GetMarkers().Entries)
	}
	return v.GoVisitor.Visit(t, p)
}
