/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/preconditions"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveUnusedGoModRequires removes `require` directives that are provably unused from the
// offline resolution: modules absent from the resolved build list and stray self-references.
type RemoveUnusedGoModRequires struct {
	recipe.Base
}

func (r *RemoveUnusedGoModRequires) Name() string {
	return "org.openrewrite.golang.migration.RemoveUnusedGoModRequires"
}

func (r *RemoveUnusedGoModRequires) DisplayName() string {
	return "Remove unused go.mod requirements"
}

func (r *RemoveUnusedGoModRequires) Description() string {
	return "Remove `require` directives that `go mod tidy` would drop, restricted to what can be proven unused from the offline resolution: modules absent from the resolved build list and stray self-references. A require present in the build list is kept even when it is neither imported nor reachable through the recorded module-graph edges, since that graph is pruned and a still-needed test-closure or build-tag-gated dependency can be unreachable in it. Uses the resolved build list attached at parse time; a no-op when that resolution did not run."
}

func (r *RemoveUnusedGoModRequires) Tags() []string { return []string{"gomod", "tidy"} }

func (r *RemoveUnusedGoModRequires) Editor() recipe.TreeVisitor {
	return preconditions.Check(
		preconditions.HasSourcePath("**/go.mod"),
		visitor.Init(&removeUnusedRequiresVisitor{}),
	)
}

type removeUnusedRequiresVisitor struct {
	visitor.GoVisitor
}

func (v *removeUnusedRequiresVisitor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	mrr := java.FindMarker[golang.GoResolutionResult](gm.Markers)
	if mrr == nil || mrr.ResolutionStatus != golang.GoResolutionResolved || len(mrr.ResolvedDependencies) == 0 {
		return gm
	}
	main := mrr.ModulePath
	buildList := buildListModules(mrr, main)

	var out []java.RightPadded[golang.GoModStatement]
	changed := false
	for _, rp := range gm.Statements {
		switch el := rp.Element.(type) {
		case *golang.GoModDirective:
			if el.Keyword == "require" && removableModule(firstValueText(el), buildList, main) {
				changed = true
				continue
			}
		case *golang.GoModBlock:
			if el.Keyword == "require" {
				kept, dropped := filterRequireBlock(el, buildList, main)
				if dropped {
					changed = true
					if len(kept.Entries) == 0 {
						continue
					}
					rp.Element = kept
				}
			}
		}
		out = append(out, rp)
	}
	if !changed {
		return gm
	}
	return gm.WithStatements(out)
}

// removableModule keeps anything in the resolved build list (the pruned graph can hide a still-needed indirect), removing only self-references and modules absent from that list.
func removableModule(modulePath string, buildList map[string]bool, main string) bool {
	if modulePath == "" || modulePath == main {
		return false
	}
	return isSelfReference(modulePath, main) || !buildList[modulePath]
}

func buildListModules(mrr *golang.GoResolutionResult, main string) map[string]bool {
	set := make(map[string]bool, len(mrr.ResolvedDependencies))
	for _, rd := range mrr.ResolvedDependencies {
		if rd.ModulePath != "" && rd.ModulePath != main {
			set[rd.ModulePath] = true
		}
	}
	return set
}

// isSelfReference reports whether modulePath names a different major version of
// the main module (e.g. `.../foo` under main `.../foo/v2`). `go mod tidy` always
// drops such a stray require, and a module importing an earlier major of itself
// under a build constraint is not a case that occurs in practice, so removing it
// stays build-safe.
func isSelfReference(modulePath, main string) bool {
	return modulePath != main && moduleBase(modulePath) == moduleBase(main)
}

// moduleBase strips a trailing `/vN` (N >= 2) major-version element from a module
// path, so two major versions of the same module share a base.
func moduleBase(modulePath string) string {
	i := strings.LastIndexByte(modulePath, '/')
	if i < 0 {
		return modulePath
	}
	last := modulePath[i+1:]
	if len(last) < 2 || last[0] != 'v' {
		return modulePath
	}
	for _, c := range last[1:] {
		if c < '0' || c > '9' {
			return modulePath
		}
	}
	if last == "v0" || last == "v1" {
		return modulePath
	}
	return modulePath[:i]
}

// filterRequireBlock drops removable entries from a require block. When the
// original first entry is dropped, the new first entry's leading newline is
// restored so the block still opens on its own line.
func filterRequireBlock(b *golang.GoModBlock, buildList map[string]bool, main string) (*golang.GoModBlock, bool) {
	var kept []java.RightPadded[golang.GoModStatement]
	dropped, firstDropped := false, false
	for i, e := range b.Entries {
		if d, ok := e.Element.(*golang.GoModDirective); ok && removableModule(firstValueText(d), buildList, main) {
			dropped = true
			if i == 0 {
				firstDropped = true
			}
			continue
		}
		kept = append(kept, e)
	}
	if !dropped {
		return b, false
	}
	if firstDropped && len(kept) > 0 {
		if d, ok := kept[0].Element.(*golang.GoModDirective); ok && !strings.HasPrefix(d.Prefix.Whitespace, "\n") {
			sp := java.Space{Whitespace: "\n" + d.Prefix.Whitespace, Comments: d.Prefix.Comments}
			kept[0].Element = d.WithPrefix(sp)
		}
	}
	return b.WithEntries(kept), true
}
