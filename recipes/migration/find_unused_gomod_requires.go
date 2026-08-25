/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// FindUnusedGoModRequires flags direct `require` directives (those without an
// `// indirect` marker) that no package in the module imports. A direct
// requirement is only justified by a direct import, so these are untidy: `go
// mod tidy` would remove them or demote them to `// indirect`.
//
// It reports rather than removes, because deciding whether such a module is
// still needed transitively requires the module graph, which is not available
// offline. FixGoModIndirectMarkers is the safe automatic counterpart (it
// demotes these to `// indirect`).
type FindUnusedGoModRequires struct {
	recipe.ScanningBase
}

func (r *FindUnusedGoModRequires) Name() string {
	return "org.openrewrite.golang.migration.FindUnusedGoModRequires"
}

func (r *FindUnusedGoModRequires) DisplayName() string {
	return "Find unused go.mod requirements"
}

func (r *FindUnusedGoModRequires) Description() string {
	return "Find direct `require` directives in go.mod that no package in the module imports. A direct requirement is only justified by a direct import, so these are candidates `go mod tidy` would remove or demote to `// indirect`. " +
		"They are reported rather than removed because deciding whether a module is still needed transitively requires the module graph, which is not available offline."
}

func (r *FindUnusedGoModRequires) Tags() []string { return []string{"gomod", "tidy", "search"} }

func (r *FindUnusedGoModRequires) InitialValue(ctx *recipe.ExecutionContext) any {
	return newImportedModulesAcc()
}

func (r *FindUnusedGoModRequires) Scanner(acc any) recipe.TreeVisitor {
	return visitor.Init(&importCollector{acc: acc.(*importedModulesAcc)})
}

func (r *FindUnusedGoModRequires) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&findUnusedRequiresEditor{acc: acc.(*importedModulesAcc)})
}

type findUnusedRequiresEditor struct {
	visitor.GoVisitor
	acc *importedModulesAcc
}

func (v *findUnusedRequiresEditor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	imported := directlyImportedModules(gm, v.acc.imports)

	changed := false
	statements := make([]java.RightPadded[golang.GoModStatement], len(gm.Statements))
	for i, rp := range gm.Statements {
		switch el := rp.Element.(type) {
		case *golang.GoModDirective:
			if el.Keyword == "require" && !hasIndirectComment(rp.After) && !imported[firstValueText(el)] {
				rp.Element = markUnused(el)
				changed = true
			}
		case *golang.GoModBlock:
			if el.Keyword == "require" {
				if marked := markUnusedBlockEntries(el, imported); marked != el {
					rp.Element = marked
					changed = true
				}
			}
		}
		statements[i] = rp
	}
	if !changed {
		return gm
	}
	return gm.WithStatements(statements)
}

func markUnusedBlockEntries(b *golang.GoModBlock, imported map[string]bool) *golang.GoModBlock {
	changed := false
	entries := make([]java.RightPadded[golang.GoModStatement], len(b.Entries))
	for i, e := range b.Entries {
		if d, ok := e.Element.(*golang.GoModDirective); ok && !hasIndirectComment(e.After) && !imported[firstValueText(d)] {
			e.Element = markUnused(d)
			changed = true
		}
		entries[i] = e
	}
	if !changed {
		return b
	}
	return b.WithEntries(entries)
}

func markUnused(d *golang.GoModDirective) *golang.GoModDirective {
	return d.WithMarkers(java.FoundSearchResult(d.Markers, "unused go.mod requirement"))
}
