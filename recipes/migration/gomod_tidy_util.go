/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// importedModulesAcc accumulates, during a scanning recipe's scan phase, the
// non-stdlib import paths seen in each .go file keyed by the file's directory,
// together with the directory of every go.mod in the run. In a multi-module
// repo those two let importsForModule scope a go.mod's imports to the files its
// own module owns, so a submodule does not count a sibling's imports as direct.
type importedModulesAcc struct {
	importsByDir map[string]map[string]struct{}
	moduleDirs   map[string]struct{}
}

func newImportedModulesAcc() *importedModulesAcc {
	return &importedModulesAcc{
		importsByDir: map[string]map[string]struct{}{},
		moduleDirs:   map[string]struct{}{},
	}
}

// importsForModule returns the union of imports from the .go files owned by the
// module rooted at moduleDir: files whose nearest enclosing go.mod directory is
// moduleDir. Files under a deeper nested module are excluded, mirroring how Go
// binds a package to its most specific module.
func (a *importedModulesAcc) importsForModule(moduleDir string) map[string]struct{} {
	imports := map[string]struct{}{}
	for fileDir, ips := range a.importsByDir {
		if a.nearestModuleDir(fileDir) != moduleDir {
			continue
		}
		for ip := range ips {
			imports[ip] = struct{}{}
		}
	}
	return imports
}

// nearestModuleDir returns the directory of the go.mod that owns files in
// fileDir: the longest module directory that contains fileDir, or "" when no
// collected module directory does.
func (a *importedModulesAcc) nearestModuleDir(fileDir string) string {
	best := ""
	bestRank := -1
	for md := range a.moduleDirs {
		if dirContains(md, fileDir) && dirRank(md) > bestRank {
			best, bestRank = md, dirRank(md)
		}
	}
	return best
}

// dirContains reports whether the directory dir contains descendant (a file in
// dir itself or in a subdirectory). Paths are slash-separated and cleaned; the
// module root "." contains everything.
func dirContains(dir, descendant string) bool {
	if dir == "." {
		return true
	}
	return descendant == dir || strings.HasPrefix(descendant, dir+"/")
}

// dirRank orders directories by depth so the nearest (deepest) module wins. The
// module root "." is the shallowest.
func dirRank(dir string) int {
	if dir == "." {
		return 0
	}
	return strings.Count(dir, "/") + 1
}

// importCollector is the scan-phase visitor shared by the go.mod tidy recipes:
// it records every non-stdlib import path per .go file directory and the
// directory of every go.mod, so imports can later be scoped per module.
type importCollector struct {
	visitor.GoVisitor
	acc *importedModulesAcc
}

func (v *importCollector) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)
	if cu.Imports == nil {
		return cu
	}
	dir := path.Dir(cu.SourcePath)
	for _, rp := range cu.Imports.Elements {
		ip := importPathOf(rp.Element)
		if ip != "" && !isStdlibImport(ip) {
			imports := v.acc.importsByDir[dir]
			if imports == nil {
				imports = map[string]struct{}{}
				v.acc.importsByDir[dir] = imports
			}
			imports[ip] = struct{}{}
		}
	}
	return cu
}

func (v *importCollector) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	v.acc.moduleDirs[path.Dir(gm.SourcePath)] = struct{}{}
	return v.GoVisitor.VisitGoMod(gm, p)
}

// importPathOf returns the unquoted import path of an import spec, or "" when
// the spec is not a plain string literal.
func importPathOf(imp *java.Import) string {
	if imp == nil {
		return ""
	}
	lit, ok := imp.Qualid.(*java.Literal)
	if !ok || lit == nil {
		return ""
	}
	raw := lit.Source
	if s, ok := lit.Value.(string); ok {
		raw = s
	}
	return strings.Trim(raw, "\"`")
}

// isStdlibImport reports whether importPath refers to a standard-library
// package. Go's own rule: a path whose first segment contains no dot is
// stdlib (real modules are hosted under a dotted domain).
func isStdlibImport(importPath string) bool {
	first := importPath
	if i := strings.IndexByte(importPath, '/'); i >= 0 {
		first = importPath[:i]
	}
	return !strings.Contains(first, ".")
}

// moduleProvides reports whether the module rooted at modulePath provides the
// package at importPath (the module path equals or is a path-boundary prefix
// of the import path).
func moduleProvides(modulePath, importPath string) bool {
	return importPath == modulePath || strings.HasPrefix(importPath, modulePath+"/")
}

// bestModuleFor returns the longest module path in candidates that provides
// importPath, or "" if none does. Longest-prefix wins so that nested modules
// bind an import to the most specific module, matching Go's package
// resolution.
func bestModuleFor(importPath string, candidates []string) string {
	best := ""
	for _, m := range candidates {
		if moduleProvides(m, importPath) && len(m) > len(best) {
			best = m
		}
	}
	return best
}

// firstValueText returns the text of a directive's first value, which for a
// require line is the module path. Returns "" when the directive has no values.
func firstValueText(d *golang.GoModDirective) string {
	if len(d.Values) == 0 {
		return ""
	}
	return d.Values[0].Text
}

// requireModulePaths returns the module path of every `require` entry in the
// go.mod (single-line and factored-block forms), and the main module path from
// the `module` directive.
func requireModulePaths(gm *golang.GoMod) (requires []string, mainModule string) {
	for _, rp := range gm.Statements {
		switch el := rp.Element.(type) {
		case *golang.GoModDirective:
			switch el.Keyword {
			case "module":
				mainModule = firstValueText(el)
			case "require":
				requires = append(requires, firstValueText(el))
			}
		case *golang.GoModBlock:
			if el.Keyword == "require" {
				for _, e := range el.Entries {
					if d, ok := e.Element.(*golang.GoModDirective); ok {
						requires = append(requires, firstValueText(d))
					}
				}
			}
		}
	}
	return requires, mainModule
}

// requiredModuleSet returns the set of module paths already declared by a
// `require` directive (single-line or block entry) in the go.mod.
func requiredModuleSet(gm *golang.GoMod) map[string]bool {
	requires, _ := requireModulePaths(gm)
	set := make(map[string]bool, len(requires))
	for _, r := range requires {
		set[r] = true
	}
	return set
}

func newIdent() uuid.UUID { return uuid.New() }

func freshMarkers() java.Markers { return java.Markers{ID: uuid.New()} }

func newGoModValue(prefix java.Space, text string) *golang.GoModValue {
	return &golang.GoModValue{Ident: uuid.New(), Prefix: prefix, Markers: freshMarkers(), Text: text}
}

// newRequireEntry builds a `<modulePath> <version>` block-entry directive with
// the given leading whitespace, adding a trailing `// indirect` comment when
// indirect. Used to insert requirements that a full `go mod tidy` would add.
func newRequireEntry(prefixWS, modulePath, version string, indirect bool) java.RightPadded[golang.GoModStatement] {
	d := &golang.GoModDirective{
		Ident:   uuid.New(),
		Prefix:  java.Space{Whitespace: prefixWS},
		Markers: freshMarkers(),
		Values: []*golang.GoModValue{
			newGoModValue(java.EmptySpace, modulePath),
			newGoModValue(java.SingleSpace, version),
		},
	}
	after := java.Space{Whitespace: "\n"}
	if indirect {
		after = withIndirectComment(after)
	}
	return java.RightPadded[golang.GoModStatement]{Element: d, After: after, Markers: freshMarkers()}
}

// directlyImportedModules returns the set of module paths declared in the
// go.mod that a package in this module imports. Each import binds to the
// longest matching module path (its nearest module); the main module absorbs
// the module's own internal imports so they never count as a dependency.
func directlyImportedModules(gm *golang.GoMod, imports map[string]struct{}) map[string]bool {
	requires, mainModule := requireModulePaths(gm)
	candidates := append(requires, mainModule)
	direct := map[string]bool{}
	for ip := range imports {
		if m := bestModuleFor(ip, candidates); m != "" && m != mainModule {
			direct[m] = true
		}
	}
	return direct
}

const indirectComment = "indirect"

// hasIndirectComment reports whether after carries a trailing `// indirect`
// comment.
func hasIndirectComment(after java.Space) bool {
	for _, c := range after.Comments {
		if strings.TrimSpace(c.Text) == indirectComment {
			return true
		}
	}
	return false
}

// withoutIndirectComment returns after with any `// indirect` comment removed,
// preserving the trailing newline that followed it and dropping the whitespace
// that preceded it on the line.
func withoutIndirectComment(after java.Space) java.Space {
	kept := make([]java.Comment, 0, len(after.Comments))
	whitespace := after.Whitespace
	for _, c := range after.Comments {
		if strings.TrimSpace(c.Text) == indirectComment {
			whitespace = strings.TrimRight(whitespace, " \t") + c.Suffix
			continue
		}
		kept = append(kept, c)
	}
	if len(kept) == 0 {
		kept = nil
	}
	return java.Space{Whitespace: whitespace, Comments: kept}
}

// withIndirectComment returns after with a trailing `// indirect` comment,
// re-homing the line's terminating newline onto the comment's suffix so the
// entry prints as `<tokens> // indirect\n`.
func withIndirectComment(after java.Space) java.Space {
	if hasIndirectComment(after) {
		return after
	}
	ws := after.Whitespace
	suffix := ""
	if i := strings.LastIndexByte(ws, '\n'); i >= 0 {
		suffix = ws[i:]
		ws = ws[:i]
	}
	ws += " "
	comment := java.Comment{Text: " " + indirectComment, Suffix: suffix}
	return java.Space{Whitespace: ws, Comments: append(append([]java.Comment{}, after.Comments...), comment)}
}
