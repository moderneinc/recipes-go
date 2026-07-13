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

// AddMissingGoModRequires adds `require` directives for modules that the
// resolved build list needs but go.mod does not yet declare — the requirements
// `go mod tidy` would add. Each module is added at its resolved version with
// the `// indirect` marker the toolchain assigned it.
//
// It reads the resolved build list from the go.mod's GoResolutionResult marker,
// which is populated at parse time by the rewrite-go toolchain resolver. When
// resolution did not run (marker has no resolved dependencies) it is a no-op.
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
	if mrr == nil {
		return gm
	}

	missing := missingRequires(gm, mrr)
	if len(missing) == 0 {
		return gm
	}
	return insertRequires(gm, missing)
}

// missingRequires returns, sorted by module path, the build-list modules that
// no `require` directive covers.
func missingRequires(gm *golang.GoMod, mrr *golang.GoResolutionResult) []missingRequire {
	required := requiredModuleSet(gm)
	seen := map[string]bool{}
	var missing []missingRequire
	for _, rd := range mrr.ResolvedDependencies {
		if rd.Main || rd.ModulePath == "" || rd.ModulePath == mrr.ModulePath {
			continue
		}
		if required[rd.ModulePath] || seen[rd.ModulePath] {
			continue
		}
		seen[rd.ModulePath] = true
		missing = append(missing, missingRequire{rd.ModulePath, rd.Version, rd.Indirect})
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i].modulePath < missing[j].modulePath })
	return missing
}

// insertRequires appends the missing requirements to the first `require` block,
// or creates a new block when none exists.
func insertRequires(gm *golang.GoMod, missing []missingRequire) *golang.GoMod {
	for i, rp := range gm.Statements {
		if b, ok := rp.Element.(*golang.GoModBlock); ok && b.Keyword == "require" {
			rp.Element = appendToRequireBlock(b, missing)
			statements := append([]java.RightPadded[golang.GoModStatement]{}, gm.Statements...)
			statements[i] = rp
			return gm.WithStatements(statements)
		}
	}

	block := newRequireBlock(missing)
	entry := java.RightPadded[golang.GoModStatement]{Element: block, After: java.Space{Whitespace: "\n"}, Markers: freshMarkers()}
	return gm.WithStatements(append(append([]java.RightPadded[golang.GoModStatement]{}, gm.Statements...), entry))
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
