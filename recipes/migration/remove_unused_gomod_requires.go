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

// RemoveUnusedGoModRequires removes `require` directives whose module provides
// no imported package and is not reachable, through the module graph, from any
// module that does — the requirements `go mod tidy` would drop.
//
// It uses the package→module map and module graph resolved at parse time
// (GoResolutionResult.PackageModules and ResolvedDependencies[].Deps). To stay
// build-safe it removes only modules unreachable from the import closure, so
// modules that merely pin a transitive version are kept. When parse-time
// resolution did not run (no PackageModules) it is a no-op: without the import
// closure it cannot tell used from unused.
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
	return "Remove `require` directives whose module provides no imported package and is unreachable through the module graph from any module that does. Uses the package→module map and module graph resolved at parse time; a no-op when that resolution did not run. Modules that pin a transitive version are kept, so the removal is build-safe."
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
	if mrr == nil || len(mrr.PackageModules) == 0 {
		return gm
	}
	needed := neededModules(mrr)
	main := mrr.ModulePath

	var out []java.RightPadded[golang.GoModStatement]
	changed := false
	for _, rp := range gm.Statements {
		switch el := rp.Element.(type) {
		case *golang.GoModDirective:
			if el.Keyword == "require" && removableModule(firstValueText(el), needed, main) {
				changed = true
				continue
			}
		case *golang.GoModBlock:
			if el.Keyword == "require" {
				kept, dropped := filterRequireBlock(el, needed, main)
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

func removableModule(modulePath string, needed map[string]bool, main string) bool {
	return modulePath != "" && modulePath != main && !needed[modulePath]
}

// neededModules returns the set of module paths that provide an imported package
// (the non-stdlib entries of PackageModules) plus everything reachable from them
// through the `go mod graph` edges recorded on ResolvedDependencies.
func neededModules(mrr *golang.GoResolutionResult) map[string]bool {
	adj := make(map[string][]string, len(mrr.ResolvedDependencies))
	for _, rd := range mrr.ResolvedDependencies {
		for _, d := range rd.Deps {
			adj[rd.ModulePath] = append(adj[rd.ModulePath], d.ModulePath)
		}
	}

	needed := map[string]bool{}
	var queue []string
	for _, pm := range mrr.PackageModules {
		if pm.Standard || pm.ModulePath == "" || needed[pm.ModulePath] {
			continue
		}
		needed[pm.ModulePath] = true
		queue = append(queue, pm.ModulePath)
	}
	for len(queue) > 0 {
		m := queue[0]
		queue = queue[1:]
		for _, n := range adj[m] {
			if !needed[n] {
				needed[n] = true
				queue = append(queue, n)
			}
		}
	}
	return needed
}

// filterRequireBlock drops removable entries from a require block. When the
// original first entry is dropped, the new first entry's leading newline is
// restored so the block still opens on its own line.
func filterRequireBlock(b *golang.GoModBlock, needed map[string]bool, main string) (*golang.GoModBlock, bool) {
	var kept []java.RightPadded[golang.GoModStatement]
	dropped, firstDropped := false, false
	for i, e := range b.Entries {
		if d, ok := e.Element.(*golang.GoModDirective); ok && removableModule(firstValueText(d), needed, main) {
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
