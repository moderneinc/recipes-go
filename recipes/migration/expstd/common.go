/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

// Package expstd migrates golang.org/x/exp/slices, /maps and /constraints to the
// standard library packages that absorbed them.
//
// x/exp/slices is now signature-identical to stdlib slices, so that one is a
// path swap. x/exp/maps is not: stdlib maps.Keys and maps.Values return an
// iter.Seq rather than a slice, and x/exp's maps.Clear became the clear builtin.
// x/exp/constraints contributed only Ordered, which became cmp.Ordered; its
// numeric constraints have no standard equivalent and block a file.
package expstd

import (
	"go/version"

	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

const (
	expModule = "golang.org/x/exp"

	expSlices      = expModule + "/slices"
	expMaps        = expModule + "/maps"
	expConstraints = expModule + "/constraints"
	expSlog        = expModule + "/slog"

	stdSlices = "slices"
	stdMaps   = "maps"
	stdCmp    = "cmp"
	stdSlog   = "log/slog"

	// slices and maps landed in the standard library in Go 1.21; the iterator
	// forms of maps.Keys and maps.Values that x/exp's slice-returning versions
	// map onto arrived in Go 1.23.
	go121 = "1.21"
	go123 = "1.23"
)

// lessFuncTakers are the x/exp/slices functions whose comparator changed shape.
// Before February 2023 each took `less func(a, b E) bool`; they now take
// `cmp func(a, b E) int`, matching the standard library. A call still passing a
// bool-returning literal is written against the old API and does not compile
// against either the current x/exp or the standard library, so the file is left
// for review rather than silently repointed.
var lessFuncTakers = map[string]bool{
	"SortFunc":       true,
	"SortStableFunc": true,
	"IsSortedFunc":   true,
	"MinFunc":        true,
	"MaxFunc":        true,
}

// slogCtxRenames map the x/exp/slog context-taking helpers to the names the
// standard library settled on. x/exp carries both spellings — the Ctx ones
// deprecated — and log/slog only the Context ones, so a call that kept the old
// name has to be renamed for the import to move.
var slogCtxRenames = map[string]string{
	"DebugCtx": "DebugContext",
	"InfoCtx":  "InfoContext",
	"WarnCtx":  "WarnContext",
	"ErrorCtx": "ErrorContext",
}

// expOnlyConstraints are the x/exp/constraints type constraints the standard
// library never adopted. Only Ordered made it, as cmp.Ordered.
var expOnlyConstraints = map[string]bool{
	"Integer":  true,
	"Float":    true,
	"Signed":   true,
	"Unsigned": true,
	"Complex":  true,
}

// moduleAcc carries the module's `go` directive from the scan phase into the
// edit phase, since a rewrite is only safe when the module targets the release
// that introduced its replacement.
type moduleAcc struct {
	goVersion string
}

func newModuleAcc() *moduleAcc { return &moduleAcc{} }

// atLeast reports whether the module targets minVersion or newer. An unknown
// version — no go.mod in the source set — is treated as unconstrained, since a
// module without a `go` directive cannot be checked and the alternative is for
// the recipe to do nothing at all.
func (a *moduleAcc) atLeast(minVersion string) bool {
	if a == nil || a.goVersion == "" {
		return true
	}
	return version.Compare("go"+a.goVersion, "go"+minVersion) >= 0
}

// scanGoVersion reads the `go` directive out of go.mod.
func scanGoVersion(acc *moduleAcc) recipe.TreeVisitor {
	return visitor.Init(&goVersionScanner{acc: acc})
}

type goVersionScanner struct {
	visitor.GoVisitor
	acc *moduleAcc
}

func (v *goVersionScanner) VisitGoModDirective(d *golang.GoModDirective, p any) java.Tree {
	if d.Keyword == "go" && len(d.Values) == 1 {
		v.acc.goVersion = d.Values[0].Text
	}
	return d
}

// bindsName reports whether cu already binds name at file scope through an
// import, which the rewrite would collide with when it introduces a reference of
// its own.
func bindsName(cu *golang.CompilationUnit, name string) bool {
	if cu == nil || cu.Imports == nil {
		return false
	}
	for _, rp := range cu.Imports.Elements {
		if pathswap.LocalName(rp.Element) == name {
			return true
		}
	}
	return false
}

// realArgs returns the arguments of mi, skipping the Empty sentinel an empty
// argument list carries.
func realArgs(mi *java.MethodInvocation) []java.Expression {
	var out []java.Expression
	for _, rp := range mi.Arguments.Elements {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		out = append(out, rp.Element)
	}
	return out
}

// qualifiedCall reports whether mi is `<pkg>.<name>(…)` for the given local
// package name, returning the called name.
func qualifiedCall(mi *java.MethodInvocation, pkg string) (string, bool) {
	if pkg == "" || mi.Name == nil || mi.Select == nil {
		return "", false
	}
	recv, ok := mi.Select.Element.(*java.Identifier)
	if !ok || recv.Name != pkg {
		return "", false
	}
	return mi.Name.Name, true
}

// qualifiedRef reports whether fa is `<pkg>.<Name>` for the given local package
// name, returning the referenced name.
func qualifiedRef(fa *java.FieldAccess, pkg string) (string, bool) {
	if pkg == "" || fa.Name.Element == nil {
		return "", false
	}
	target, ok := fa.Target.(*java.Identifier)
	if !ok || target.Name != pkg {
		return "", false
	}
	return fa.Name.Element.Name, true
}
