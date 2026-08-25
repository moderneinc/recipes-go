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

// FixGoModIndirectMarkers corrects the `// indirect` markers on `require`
// directives in go.mod. A requirement is direct when a package in the module
// imports it and indirect otherwise, exactly as `go mod tidy` determines it.
//
// This is a scanning recipe: it collects the set of imported packages across
// every .go file in the module, then rewrites the go.mod markers accordingly.
//
// It never removes a `require`, so it is build-safe offline: a requirement that
// is genuinely unused is conservatively marked `// indirect` rather than
// removed (removal needs the transitive module graph, which is not available
// offline). See FindUnusedGoModRequires for reporting removal candidates.
type FixGoModIndirectMarkers struct {
	recipe.ScanningBase
}

func (r *FixGoModIndirectMarkers) Name() string {
	return "org.openrewrite.golang.migration.FixGoModIndirectMarkers"
}

func (r *FixGoModIndirectMarkers) DisplayName() string {
	return "Fix go.mod `// indirect` markers"
}

func (r *FixGoModIndirectMarkers) Description() string {
	return "Correct the `// indirect` markers on `require` directives in go.mod: a requirement is direct when a package in the module imports it and indirect otherwise. " +
		"Requirements are never removed, so the change is always build-safe; a genuinely unused requirement is marked `// indirect` rather than removed."
}

func (r *FixGoModIndirectMarkers) Tags() []string { return []string{"gomod", "tidy"} }

func (r *FixGoModIndirectMarkers) InitialValue(ctx *recipe.ExecutionContext) any {
	return newImportedModulesAcc()
}

func (r *FixGoModIndirectMarkers) Scanner(acc any) recipe.TreeVisitor {
	return visitor.Init(&importCollector{acc: acc.(*importedModulesAcc)})
}

func (r *FixGoModIndirectMarkers) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&fixIndirectEditor{acc: acc.(*importedModulesAcc)})
}

type fixIndirectEditor struct {
	visitor.GoVisitor
	acc *importedModulesAcc
}

func (v *fixIndirectEditor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	directImported := directlyImportedModules(gm, v.acc.imports)

	changed := false
	statements := make([]java.RightPadded[golang.GoModStatement], len(gm.Statements))
	for i, rp := range gm.Statements {
		switch el := rp.Element.(type) {
		case *golang.GoModDirective:
			if el.Keyword == "require" {
				if after := setIndirect(rp.After, !directImported[firstValueText(el)]); !java.SpaceEqual(after, rp.After) {
					rp.After = after
					changed = true
				}
			}
		case *golang.GoModBlock:
			if el.Keyword == "require" {
				if fixed := fixRequireBlock(el, directImported); fixed != el {
					rp.Element = fixed
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

func fixRequireBlock(b *golang.GoModBlock, directImported map[string]bool) *golang.GoModBlock {
	changed := false
	entries := make([]java.RightPadded[golang.GoModStatement], len(b.Entries))
	for i, e := range b.Entries {
		if d, ok := e.Element.(*golang.GoModDirective); ok {
			if after := setIndirect(e.After, !directImported[firstValueText(d)]); !java.SpaceEqual(after, e.After) {
				e.After = after
				changed = true
			}
		}
		entries[i] = e
	}
	if !changed {
		return b
	}
	return b.WithEntries(entries)
}

func setIndirect(after java.Space, indirect bool) java.Space {
	if indirect {
		return withIndirectComment(after)
	}
	return withoutIndirectComment(after)
}
