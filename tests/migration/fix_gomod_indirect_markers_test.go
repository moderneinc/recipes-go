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

// runFixIndirect drives the scan-then-edit lifecycle of a ScanningRecipe by
// hand: the Go unit-test harness (RewriteRun) only invokes Editor(), so a
// cross-file scanning recipe has to be exercised the way the production RPC
// server does — scan every .go file, then run the go.mod editor.
func runFixIndirect(t *testing.T, goMod string, goFiles ...string) string {
	t.Helper()
	r := &migration.FixGoModIndirectMarkers{}
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
	return printer.PrintGoMod(result.(*golang.GoMod))
}

func TestFixGoModIndirectStripsMarkerFromImportedModule(t *testing.T) {
	// given a require marked // indirect that the module actually imports
	goMod := `
		module example.com/app

		go 1.22

		require (
			github.com/foo/bar v1.2.3 // indirect
			github.com/baz/qux v1.0.0 // indirect
		)
	`
	goFile := `
		package main

		import "github.com/foo/bar"

		func main() { _ = bar.A }
	`

	// when
	got := runFixIndirect(t, goMod, goFile)

	// then the imported one becomes direct, the unused one stays indirect
	want := test.TrimIndent(`
		module example.com/app

		go 1.22

		require (
			github.com/foo/bar v1.2.3
			github.com/baz/qux v1.0.0 // indirect
		)
	`)
	if got != want {
		t.Errorf("mismatch\n\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestFixGoModIndirectAddsMarkerToUnusedModule(t *testing.T) {
	// given a direct require that nothing imports
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

		import "github.com/foo/bar/sub"

		func main() { _ = sub.A }
	`

	// when
	got := runFixIndirect(t, goMod, goFile)

	// then the unimported module gains // indirect; the sub-package import keeps its module direct
	want := test.TrimIndent(`
		module example.com/app

		go 1.22

		require (
			github.com/foo/bar v1.2.3
			github.com/baz/qux v1.0.0 // indirect
		)
	`)
	if got != want {
		t.Errorf("mismatch\n\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestFixGoModIndirectSingleLineRequire(t *testing.T) {
	// given a single-line require that is imported
	goMod := `
		module example.com/app

		go 1.22

		require github.com/foo/bar v1.2.3 // indirect
	`
	goFile := `
		package main

		import "github.com/foo/bar"

		func main() { _ = bar.A }
	`

	// when
	got := runFixIndirect(t, goMod, goFile)

	// then
	want := test.TrimIndent(`
		module example.com/app

		go 1.22

		require github.com/foo/bar v1.2.3
	`)
	if got != want {
		t.Errorf("mismatch\n\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestFixGoModIndirectNestedModulesBindToLongestPath(t *testing.T) {
	// given a parent and a nested child module, importing only the child
	goMod := `
		module example.com/app

		go 1.22

		require (
			github.com/foo/bar v1.2.3
			github.com/foo/bar/nested v1.0.0
		)
	`
	goFile := `
		package main

		import "github.com/foo/bar/nested/pkg"

		func main() { _ = pkg.A }
	`

	// when
	got := runFixIndirect(t, goMod, goFile)

	// then the import binds to the nested module only; the parent becomes indirect
	want := test.TrimIndent(`
		module example.com/app

		go 1.22

		require (
			github.com/foo/bar v1.2.3 // indirect
			github.com/foo/bar/nested v1.0.0
		)
	`)
	if got != want {
		t.Errorf("mismatch\n\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestFixGoModIndirectNoChangeWhenAlreadyCorrect(t *testing.T) {
	// given a go.mod whose markers already match usage
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
	got := runFixIndirect(t, goMod, goFile)

	// then output is byte-identical to input
	want := test.TrimIndent(goMod)
	if got != want {
		t.Errorf("expected no change\n\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}
