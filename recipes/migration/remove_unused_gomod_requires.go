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

// RemoveUnusedGoModRequires removes `// indirect` requires unreachable from the imported-package
// modules and retained direct requires (kept as possible gated imports), plus stray self-references.
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
	return "Remove `require` directives that `go mod tidy` would drop, restricted to what can be proven unused offline: unreachable `// indirect` requires and stray self-references. Direct requires are kept even when unimported in the scanned build configuration, since their import may be gated behind an inactive build constraint. Uses the package→module map and module graph resolved at parse time; a no-op when that resolution did not run."
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
	if mrr == nil || mrr.ResolutionStatus != golang.GoResolutionResolved || len(mrr.PackageModules) == 0 {
		return gm
	}
	main := mrr.ModulePath
	needed := neededModules(mrr, main, retainedDirectRequires(gm, main))

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

// retainedDirectRequires returns the module paths of the direct requires that
// are always kept: every `require` not marked `// indirect`, except a stray
// self-reference to another major version of the main module. These seed the
// needed set so a build-constraint-gated direct import — invisible to the
// primary-configuration scan — is not deleted, and so its indirect closure
// stays reachable.
func retainedDirectRequires(gm *golang.GoMod, main string) map[string]bool {
	direct := map[string]bool{}
	add := func(d *golang.GoModDirective, after java.Space) {
		modulePath := firstValueText(d)
		if modulePath == "" || hasIndirectComment(after) || isSelfReference(modulePath, main) {
			return
		}
		direct[modulePath] = true
	}
	for _, rp := range gm.Statements {
		switch el := rp.Element.(type) {
		case *golang.GoModDirective:
			if el.Keyword == "require" {
				add(el, rp.After)
			}
		case *golang.GoModBlock:
			if el.Keyword == "require" {
				for _, e := range el.Entries {
					if d, ok := e.Element.(*golang.GoModDirective); ok {
						add(d, e.After)
					}
				}
			}
		}
	}
	return direct
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

func neededModules(mrr *golang.GoResolutionResult, main string, directSeeds map[string]bool) map[string]bool {
	adj := make(map[string][]string, len(mrr.ResolvedDependencies))
	for _, rd := range mrr.ResolvedDependencies {
		for _, d := range rd.Deps {
			adj[rd.ModulePath] = append(adj[rd.ModulePath], d.ModulePath)
		}
	}

	needed := map[string]bool{}
	var queue []string
	seed := func(modulePath string) {
		if modulePath == "" || modulePath == main || needed[modulePath] {
			return
		}
		needed[modulePath] = true
		queue = append(queue, modulePath)
	}
	for _, pm := range mrr.PackageModules {
		if pm.Standard {
			continue
		}
		seed(pm.ModulePath)
	}
	for m := range directSeeds {
		seed(m)
	}
	for len(queue) > 0 {
		m := queue[0]
		queue = queue[1:]
		for _, n := range adj[m] {
			if n != main && !needed[n] {
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
