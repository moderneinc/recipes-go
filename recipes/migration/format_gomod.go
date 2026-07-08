/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"sort"
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// FormatGoMod canonicalizes go.mod formatting the way `go mod tidy` does, to
// the extent that is safe offline: it sorts the entries of each factored
// `require ( … )` block by module path. Each entry keeps its own version and
// `// indirect` marker; only the ordering changes.
//
// It does not yet merge multiple require blocks or split direct from indirect
// requirements into separate blocks — those structural rewrites are left for a
// follow-up. Blocks whose entries carry their own line comments are left
// untouched to avoid moving a comment away from the line it documents.
type FormatGoMod struct {
	recipe.Base
}

func (r *FormatGoMod) Name() string { return "org.openrewrite.golang.migration.FormatGoMod" }

func (r *FormatGoMod) DisplayName() string { return "Format go.mod" }

func (r *FormatGoMod) Description() string {
	return "Sort the entries of each factored `require ( … )` block in go.mod by module path, matching `go mod tidy` ordering. Versions and `// indirect` markers travel with their entry; only the ordering changes."
}

func (r *FormatGoMod) Tags() []string { return []string{"gomod", "tidy"} }

func (r *FormatGoMod) Editor() recipe.TreeVisitor {
	return visitor.Init(&formatGoModVisitor{})
}

type formatGoModVisitor struct {
	visitor.GoVisitor
}

func (v *formatGoModVisitor) VisitGoModBlock(b *golang.GoModBlock, p any) java.Tree {
	b = v.GoVisitor.VisitGoModBlock(b, p).(*golang.GoModBlock)
	if b.Keyword != "require" {
		return b
	}
	return sortRequireBlock(b)
}

// sortRequireBlock returns b with its entries ordered by module path. It is a
// no-op unless the block is in the standard one-entry-per-line shape: every
// entry is a directive, the first entry's prefix starts the line (contains a
// newline), and no entry prefix carries a comment. Reordering reassigns the
// per-position leading whitespace (the first line has a newline before its
// indent, the rest only the indent) while each entry keeps its own trailing
// space, so the `// indirect` comment stays with its module.
func sortRequireBlock(b *golang.GoModBlock) *golang.GoModBlock {
	if len(b.Entries) < 2 {
		return b
	}

	directives := make([]*golang.GoModDirective, len(b.Entries))
	for i, e := range b.Entries {
		d, ok := e.Element.(*golang.GoModDirective)
		if !ok || len(d.Prefix.Comments) > 0 {
			return b
		}
		directives[i] = d
	}
	if !strings.Contains(directives[0].Prefix.Whitespace, "\n") {
		return b
	}

	firstPrefix := directives[0].Prefix
	restPrefix := directives[1].Prefix

	sorted := make([]java.RightPadded[golang.GoModStatement], len(b.Entries))
	copy(sorted, b.Entries)
	if sort.SliceIsSorted(sorted, func(i, j int) bool { return entryModulePath(sorted[i]) < entryModulePath(sorted[j]) }) {
		return b
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		return entryModulePath(sorted[i]) < entryModulePath(sorted[j])
	})

	for i, e := range sorted {
		d := e.Element.(*golang.GoModDirective)
		if i == 0 {
			d = d.WithPrefix(firstPrefix)
		} else {
			d = d.WithPrefix(restPrefix)
		}
		e.Element = d
		sorted[i] = e
	}
	return b.WithEntries(sorted)
}

func entryModulePath(e java.RightPadded[golang.GoModStatement]) string {
	if d, ok := e.Element.(*golang.GoModDirective); ok {
		return firstValueText(d)
	}
	return ""
}
