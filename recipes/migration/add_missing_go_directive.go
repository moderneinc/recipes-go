/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"runtime"
	"strings"

	goversion "go/version"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/preconditions"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

const fallbackGoDirectiveVersion = "1.16"

func DefaultGoDirectiveVersion() string {
	v := strings.TrimPrefix(runtime.Version(), "go")
	if goversion.IsValid("go" + v) {
		return v
	}
	return fallbackGoDirectiveVersion
}

type AddMissingGoDirective struct {
	recipe.Base
	GoVersion string
}

func (r *AddMissingGoDirective) Name() string {
	return "org.openrewrite.golang.migration.AddMissingGoDirective"
}

func (r *AddMissingGoDirective) DisplayName() string { return "Add a missing `go` directive" }

func (r *AddMissingGoDirective) Description() string {
	return "Add a `go` directive to a go.mod that has none, as `go mod tidy` always writes one. Defaults to the toolchain version running the recipe, the version `go mod tidy` would record."
}

func (r *AddMissingGoDirective) Tags() []string { return []string{"gomod", "tidy"} }

func (r *AddMissingGoDirective) Options() []recipe.OptionDescriptor {
	return []recipe.OptionDescriptor{
		recipe.Option("goVersion", "Go version",
			"The version to set on the added `go` directive, e.g. `1.24`. Defaults to the toolchain version running the recipe.").
			WithExample("1.24").
			WithValue(r.GoVersion),
	}
}

func (r *AddMissingGoDirective) Editor() recipe.TreeVisitor {
	version := r.GoVersion
	if version == "" {
		version = DefaultGoDirectiveVersion()
	}
	return preconditions.Check(
		preconditions.HasSourcePath("**/go.mod"),
		visitor.Init(&addMissingGoDirectiveVisitor{version: version}),
	)
}

type addMissingGoDirectiveVisitor struct {
	visitor.GoVisitor
	version string
}

func (v *addMissingGoDirectiveVisitor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	moduleIdx := -1
	for i, rp := range gm.Statements {
		d, ok := rp.Element.(*golang.GoModDirective)
		if !ok {
			continue
		}
		if d.Keyword == "go" {
			return gm
		}
		if d.Keyword == "module" {
			moduleIdx = i
		}
	}

	directive := &golang.GoModDirective{
		Ident:   newIdent(),
		Prefix:  java.Space{Whitespace: "\n"},
		Markers: freshMarkers(),
		Keyword: "go",
		Values:  []*golang.GoModValue{newGoModValue(java.SingleSpace, v.version)},
	}
	entry := java.RightPadded[golang.GoModStatement]{Element: directive, After: java.Space{Whitespace: "\n"}, Markers: freshMarkers()}

	insertAt := moduleIdx + 1
	statements := make([]java.RightPadded[golang.GoModStatement], 0, len(gm.Statements)+1)
	statements = append(statements, gm.Statements[:insertAt]...)
	statements = append(statements, entry)
	statements = append(statements, gm.Statements[insertAt:]...)
	return gm.WithStatements(statements)
}
