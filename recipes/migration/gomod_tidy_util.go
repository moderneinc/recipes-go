/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// importedModulesAcc accumulates the third-party import paths seen across all
// .go files in a module during a scanning recipe's scan phase.
type importedModulesAcc struct {
	imports map[string]struct{}
}

func newImportedModulesAcc() *importedModulesAcc {
	return &importedModulesAcc{imports: map[string]struct{}{}}
}

// importCollector is the scan-phase visitor shared by the go.mod tidy recipes:
// it records every non-stdlib import path across the module's .go files.
type importCollector struct {
	visitor.GoVisitor
	acc *importedModulesAcc
}

func (v *importCollector) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)
	if cu.Imports == nil {
		return cu
	}
	for _, rp := range cu.Imports.Elements {
		ip := importPathOf(rp.Element)
		if ip != "" && !isStdlibImport(ip) {
			v.acc.imports[ip] = struct{}{}
		}
	}
	return cu
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

const indirectComment = "// indirect"

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
	comment := java.Comment{Kind: java.LineComment, Text: indirectComment, Suffix: suffix}
	return java.Space{Whitespace: ws, Comments: append(append([]java.Comment{}, after.Comments...), comment)}
}
