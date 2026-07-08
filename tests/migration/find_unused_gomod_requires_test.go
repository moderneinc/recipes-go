/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration_test

import (
	"fmt"
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/test"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
)

func runFindUnused(t *testing.T, goMod string, goFiles ...string) string {
	t.Helper()
	r := &migration.FindUnusedGoModRequires{}
	ctx := recipe.NewExecutionContext()
	acc := r.InitialValue(ctx)

	scanner := r.Scanner(acc)
	p := parser.NewGoParser()
	for i, f := range goFiles {
		cu, err := p.Parse(fmt.Sprintf("f%d.go", i), test.TrimIndent(f))
		if err != nil {
			t.Fatalf("parse .go: %v", err)
		}
		scanner.Visit(cu, ctx)
	}

	gm, err := parser.ParseGoModFile("go.mod", test.TrimIndent(goMod))
	if err != nil {
		t.Fatalf("parse go.mod: %v", err)
	}
	result := r.EditorWithData(acc).Visit(gm, ctx)
	return printer.PrintGoModWithMarkers(result.(*golang.GoMod), printer.DefaultMarkerPrinter)
}

func TestFindUnusedGoModRequiresFlagsDirectUnimported(t *testing.T) {
	// given a direct require that nothing imports, alongside an imported one
	goMod := `
		module example.com/app

		go 1.22

		require (
			github.com/foo/bar v1.2.3
			github.com/baz/qux v1.0.0
		)
	`
	goFile := `
		package main

		import "github.com/foo/bar"

		func main() { _ = bar.A }
	`

	// when
	got := runFindUnused(t, goMod, goFile)

	// then only the unimported direct require is flagged
	want := test.TrimIndent(`
		module example.com/app

		go 1.22

		require (
			github.com/foo/bar v1.2.3
			/*~~(unused go.mod requirement)~~>*/github.com/baz/qux v1.0.0
		)
	`)
	if got != want {
		t.Errorf("mismatch\n\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestFindUnusedGoModRequiresIgnoresIndirect(t *testing.T) {
	// given an unimported require that is already marked // indirect
	goMod := `
		module example.com/app

		go 1.22

		require (
			github.com/foo/bar v1.2.3
			github.com/baz/qux v1.0.0 // indirect
		)
	`
	goFile := `
		package main

		import "github.com/foo/bar"

		func main() { _ = bar.A }
	`

	// when
	got := runFindUnused(t, goMod, goFile)

	// then nothing is flagged (indirect requires are not reported)
	want := test.TrimIndent(goMod)
	if got != want {
		t.Errorf("expected no marks\n\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestFindUnusedGoModRequiresSingleLine(t *testing.T) {
	// given a single-line direct require that nothing imports
	goMod := `
		module example.com/app

		go 1.22

		require github.com/baz/qux v1.0.0
	`

	// when
	got := runFindUnused(t, goMod)

	// then
	want := test.TrimIndent(`
		module example.com/app

		go 1.22

		/*~~(unused go.mod requirement)~~>*/require github.com/baz/qux v1.0.0
	`)
	if got != want {
		t.Errorf("mismatch\n\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}
