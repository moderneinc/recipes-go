/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package tests

import (
	"fmt"
	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Recipes whose output still carries a bare call, and why each is reported
// rather than failed.
var attributionGaps = map[string]string{
	"PreferBytesBufferString": "rewrite-go pkg/template types no call whose receiver is a capture",
	"MigrateToJSONV2":         "attributed by hand on branch re-attribute-types-in-jsonv2-recipes",
	"PreserveV1Semantics":     "attributed by hand on branch re-attribute-types-in-jsonv2-recipes",
}

// Go builtins, type conversions and IIFEs, which the parser leaves with a nil
// MethodType. A recipe emitting one is at parity with parsed source.
var untypedByParser = map[string]bool{
	"": true, "append": true, "cap": true, "clear": true, "close": true,
	"complex": true, "copy": true, "delete": true, "imag": true, "len": true,
	"make": true, "max": true, "min": true, "new": true, "panic": true,
	"print": true, "println": true, "real": true, "recover": true,
	"any": true, "bool": true, "byte": true, "error": true, "float32": true,
	"float64": true, "int": true, "int8": true, "int16": true, "int32": true,
	"int64": true, "rune": true, "string": true, "uint": true, "uint8": true,
	"uint16": true, "uint32": true, "uint64": true, "uintptr": true,
}

// TestRewrittenTreesStayAttributed runs every registered recipe over the source
// snippets the suite already carries and reports each call the recipe emitted
// without the type a parsed one would have had. RewriteRun compares printed
// source, so an unattributed rewrite passes its own test and surfaces only here.
// See CLAUDE.md: Type Attribution.
func TestRewrittenTreesStayAttributed(t *testing.T) {
	byType := map[string][]recipe.Recipe{}
	for _, r := range registeredRecipes() {
		byType[reflect.TypeOf(r).Elem().Name()] = append(byType[reflect.TypeOf(r).Elem().Name()], r)
	}

	failures := map[string]map[string]bool{}
	for _, sn := range recipeSnippets(t) {
		cu, err := parser.NewGoParser().Parse("main.go", test.TrimIndent(strings.TrimPrefix(sn.src, "\n")))
		if err != nil {
			continue
		}
		parsed := bareCalls(cu)
		for _, typeName := range sn.recipes {
			for _, r := range byType[typeName] {
				after, err := runToCompletion(r, cu)
				if err != nil {
					t.Errorf("%s panicked on %s: %v", typeName, sn.file, err)
					continue
				}
				for call := range bareCalls(after) {
					// A call the snippet already had bare is the parser's doing.
					if parsed[call] {
						continue
					}
					if failures[typeName] == nil {
						failures[typeName] = map[string]bool{}
					}
					failures[typeName][call+"  ["+sn.file+"]"] = true
				}
			}
		}
	}

	for _, typeName := range sortedKeys(failures) {
		if reason, known := attributionGaps[typeName]; known {
			t.Logf("known gap in %s (%s):\n    %s", typeName, reason, strings.Join(sortedKeys(failures[typeName]), "\n    "))
			continue
		}
		t.Errorf("%s emits a call the parser would have typed:\n    %s", typeName, strings.Join(sortedKeys(failures[typeName]), "\n    "))
	}
}

// bareCalls returns one entry per call in the tree that is missing a type the
// parser supplies: the call's own method type, its package qualifier's type, or
// the type of an identifier passed straight through as an argument.
func bareCalls(tree java.Tree) map[string]bool {
	c := visitor.Init(&bareCallCollector{found: map[string]bool{}})
	c.Visit(tree, nil)
	return c.found
}

type bareCallCollector struct {
	visitor.GoVisitor
	found map[string]bool
}

func (v *bareCallCollector) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	name := ""
	if mi.Name != nil {
		name = mi.Name.Name
	}
	var qualifier *java.Identifier
	if mi.Select != nil {
		qualifier, _ = mi.Select.Element.(*java.Identifier)
	}
	call := name
	if qualifier != nil {
		call = qualifier.Name + "." + name
	}

	if qualifier == nil && untypedByParser[name] {
		return v.GoVisitor.VisitMethodInvocation(mi, p)
	}
	switch {
	case mi.MethodType == nil:
		v.found[fmt.Sprintf("%s: no method type", call)] = true
	case mi.MethodType.DeclaringType == nil:
		v.found[fmt.Sprintf("%s: method type declares no type", call)] = true
	}
	if qualifier != nil && qualifier.Type == nil {
		v.found[fmt.Sprintf("%s: qualifier %s has no type", call, qualifier.Name)] = true
	}
	for _, arg := range mi.Arguments.Elements {
		if id, ok := arg.Element.(*java.Identifier); ok && id.Type == nil {
			v.found[fmt.Sprintf("%s: argument %s has no type", call, id.Name)] = true
		}
	}
	return v.GoVisitor.VisitMethodInvocation(mi, p)
}

// runToCompletion applies a recipe the way a run would — scan, edit, then the
// visitors it queued — and returns the resulting tree.
func runToCompletion(r recipe.Recipe, tree java.Tree) (out java.Tree, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%v", p)
		}
	}()
	ctx := recipe.NewExecutionContext()
	for _, each := range append([]recipe.Recipe{r}, r.RecipeList()...) {
		for _, editor := range editorsOf(each, ctx) {
			if next := editor.Visit(tree, ctx); next != nil {
				tree = next
			}
			tree = visitor.DrainAfterVisits(editor, tree, ctx)
		}
	}
	return tree, nil
}

// editorsOf returns the visitors that make up one recipe's pass over a file: the
// scanner and data-driven editor for a scanning recipe, the plain editor
// otherwise.
func editorsOf(r recipe.Recipe, ctx *recipe.ExecutionContext) []recipe.TreeVisitor {
	if scanning, ok := r.(recipe.ScanningRecipe); ok {
		acc := scanning.InitialValue(ctx)
		return nonNil(scanning.Scanner(acc), scanning.EditorWithData(acc))
	}
	return nonNil(r.Editor())
}

func nonNil(visitors ...recipe.TreeVisitor) []recipe.TreeVisitor {
	var out []recipe.TreeVisitor
	for _, v := range visitors {
		if v != nil {
			out = append(out, v)
		}
	}
	return out
}

// snippet is one Go source sample from a test file, tagged with the recipe types
// that file names.
type snippet struct {
	file    string
	src     string
	recipes []string
}

// recipeSnippets harvests the corpus the suite already maintains: every raw Go
// source literal in tests/, attributed to the recipes its own file exercises.
func recipeSnippets(t *testing.T) []snippet {
	t.Helper()
	var out []snippet
	fset := gotoken.NewFileSet()
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, parseErr := goparser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil
		}
		names := map[string]bool{}
		var sources []string
		goast.Inspect(file, func(n goast.Node) bool {
			switch x := n.(type) {
			case *goast.CompositeLit:
				if sel, ok := x.Type.(*goast.SelectorExpr); ok {
					names[sel.Sel.Name] = true
				}
			case *goast.BasicLit:
				if strings.HasPrefix(x.Value, "`") {
					if v, e := strconv.Unquote(x.Value); e == nil && strings.Contains(v, "package ") {
						sources = append(sources, v)
					}
				}
			}
			return true
		})
		recipes := sortedKeys(names)
		for _, src := range sources {
			out = append(out, snippet{file: path, src: src, recipes: recipes})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func sortedKeys(m any) []string {
	var keys []string
	switch typed := m.(type) {
	case map[string]bool:
		for k := range typed {
			keys = append(keys, k)
		}
	case map[string]map[string]bool:
		for k := range typed {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}
