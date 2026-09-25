/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"sort"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/preconditions"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AddMissingGoModRequires adds `require` directives for modules that provide an
// imported package but go.mod does not yet declare — the requirements `go mod
// tidy` would add. Each module is added at its resolved version with the
// `// indirect` marker the toolchain assigned it.
//
// It reads the resolved build list and package→module map from the go.mod's
// GoResolutionResult marker, populated at parse time by the rewrite-go toolchain
// resolver. Only modules named in the package→module map are added: under module
// graph pruning (go >=1.17) the full build list carries transitively-reachable
// modules whose packages are never imported (e.g. a dep of an unimported package
// of a dependency), and `go mod tidy` does not record those. It acts only when
// the marker's ResolutionStatus is RESOLVED and a package→module map is present;
// any other status or a missing map means it cannot tell imported from merely
// reachable, so it is a no-op.
type AddMissingGoModRequires struct {
	recipe.Base
}

func (r *AddMissingGoModRequires) Name() string {
	return "org.openrewrite.golang.migration.AddMissingGoModRequires"
}

func (r *AddMissingGoModRequires) DisplayName() string {
	return "Add missing go.mod requirements"
}

func (r *AddMissingGoModRequires) Description() string {
	return "Add `require` directives for modules the resolved build list needs but go.mod does not declare, at their resolved versions and with the `// indirect` marker the toolchain assigned. Mirrors what `go mod tidy` adds, using the module graph resolved at parse time."
}

func (r *AddMissingGoModRequires) Tags() []string { return []string{"gomod", "tidy"} }

func (r *AddMissingGoModRequires) Editor() recipe.TreeVisitor {
	return preconditions.Check(
		preconditions.HasSourcePath("**/go.mod"),
		visitor.Init(&addMissingRequiresVisitor{}),
	)
}

type addMissingRequiresVisitor struct {
	visitor.GoVisitor
}

type missingRequire struct {
	modulePath string
	version    string
	indirect   bool
}

func (v *addMissingRequiresVisitor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	mrr := java.FindMarker[golang.GoResolutionResult](gm.Markers)
	if mrr == nil || mrr.ResolutionStatus != golang.GoResolutionResolved || len(mrr.PackageModules) == 0 {
		return gm
	}

	missing := missingRequires(gm, mrr)
	if len(missing) == 0 {
		return gm
	}
	return insertRequires(gm, missing)
}

// missingRequires returns, sorted by module path, the build-list modules that
// provide an imported package but no `require` directive covers.
func missingRequires(gm *golang.GoMod, mrr *golang.GoResolutionResult) []missingRequire {
	required := requiredModuleSet(gm)
	imported := importedModules(mrr)
	seen := map[string]bool{}
	var missing []missingRequire
	for _, rd := range mrr.ResolvedDependencies {
		if rd.Main || rd.ModulePath == "" || rd.ModulePath == mrr.ModulePath {
			continue
		}
		if required[rd.ModulePath] || seen[rd.ModulePath] || !imported[rd.ModulePath] {
			continue
		}
		seen[rd.ModulePath] = true
		missing = append(missing, missingRequire{rd.ModulePath, rd.Version, rd.Indirect})
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i].modulePath < missing[j].modulePath })
	return missing
}

// importedModules returns the set of non-stdlib module paths that provide an
// imported package, from the parse-time package→module map.
func importedModules(mrr *golang.GoResolutionResult) map[string]bool {
	imported := map[string]bool{}
	for _, pm := range mrr.PackageModules {
		if pm.Standard || pm.ModulePath == "" {
			continue
		}
		imported[pm.ModulePath] = true
	}
	return imported
}

// AddRequire returns gm with a `require modulePath version` directive added when
// no `require` for modulePath is already present; otherwise gm is returned
// unchanged. A direct entry is routed to the direct `require` block and an
// indirect one to the `// indirect` block, mirroring how `go mod tidy` keeps
// direct and indirect requirements separated; a suitable block is created when
// none exists. It does not touch go.sum, so callers introducing a brand-new
// module still need a `go mod tidy` / `go mod download` to complete resolution.
func AddRequire(gm *golang.GoMod, modulePath, version string, indirect bool) *golang.GoMod {
	if requiredModuleSet(gm)[modulePath] {
		return gm
	}
	return insertRequires(gm, []missingRequire{{modulePath: modulePath, version: version, indirect: indirect}})
}

// insertRequires adds the missing requirements, routing direct entries to the
// direct `require` block and indirect entries to the `// indirect` block so the
// two stay separated the way `go mod tidy` keeps them. A block is created when no
// suitable one exists. Existing entries are never reordered or re-split.
func insertRequires(gm *golang.GoMod, missing []missingRequire) *golang.GoMod {
	var direct, indirect []missingRequire
	for _, m := range missing {
		if m.indirect {
			indirect = append(indirect, m)
		} else {
			direct = append(direct, m)
		}
	}

	statements := append([]java.RightPadded[golang.GoModStatement]{}, gm.Statements...)
	statements = routeToRequireBlock(statements, direct, false)
	statements = routeToRequireBlock(statements, indirect, true)
	return gm.WithStatements(statements)
}

// routeToRequireBlock appends entries to the require block matching their
// directness — an all-`// indirect` block for indirect entries, a direct or
// mixed block for direct ones — creating a new block when none matches.
func routeToRequireBlock(statements []java.RightPadded[golang.GoModStatement], entries []missingRequire, indirect bool) []java.RightPadded[golang.GoModStatement] {
	if len(entries) == 0 {
		return statements
	}
	if i := findRequireBlock(statements, indirect); i >= 0 {
		rp := statements[i]
		rp.Element = appendToRequireBlock(rp.Element.(*golang.GoModBlock), entries)
		statements[i] = rp
		return statements
	}
	block := newRequireBlock(entries)
	entry := java.RightPadded[golang.GoModStatement]{Element: block, After: java.Space{Whitespace: "\n"}, Markers: freshMarkers()}
	return append(statements, entry)
}

// findRequireBlock returns the index of the require block that should receive an
// entry of the given directness: the first all-`// indirect` block for indirect
// entries, or the first block that is not all-indirect for direct entries.
// Returns -1 when no such block exists.
func findRequireBlock(statements []java.RightPadded[golang.GoModStatement], indirect bool) int {
	for i, rp := range statements {
		if b, ok := rp.Element.(*golang.GoModBlock); ok && b.Keyword == "require" && isAllIndirectBlock(b) == indirect {
			return i
		}
	}
	return -1
}

// isAllIndirectBlock reports whether every entry in a non-empty require block
// carries the `// indirect` marker, marking it as the block go.mod dedicates to
// indirect requirements.
func isAllIndirectBlock(b *golang.GoModBlock) bool {
	if len(b.Entries) == 0 {
		return false
	}
	for _, e := range b.Entries {
		if _, ok := e.Element.(*golang.GoModDirective); !ok || !hasIndirectComment(e.After) {
			return false
		}
	}
	return true
}

func appendToRequireBlock(b *golang.GoModBlock, missing []missingRequire) *golang.GoModBlock {
	indent := "\t"
	if len(b.Entries) > 0 {
		if d, ok := b.Entries[0].Element.(*golang.GoModDirective); ok {
			indent = d.Prefix.Indent()
		}
	}
	entries := append([]java.RightPadded[golang.GoModStatement]{}, b.Entries...)
	for _, m := range missing {
		entries = append(entries, newRequireEntry(indent, m.modulePath, m.version, m.indirect))
	}
	return b.WithEntries(entries)
}

func newRequireBlock(missing []missingRequire) *golang.GoModBlock {
	entries := make([]java.RightPadded[golang.GoModStatement], len(missing))
	for i, m := range missing {
		prefix := "\t"
		if i == 0 {
			prefix = "\n\t"
		}
		entries[i] = newRequireEntry(prefix, m.modulePath, m.version, m.indirect)
	}
	return &golang.GoModBlock{
		Ident:        newIdent(),
		Prefix:       java.Space{Whitespace: "\n"},
		Markers:      freshMarkers(),
		Keyword:      "require",
		BeforeLParen: java.SingleSpace,
		Entries:      entries,
		BeforeRParen: java.EmptySpace,
	}
}
