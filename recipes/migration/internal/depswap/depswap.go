/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

// Package depswap keeps a go.mod `require` in step with a module path migration
// happening in the source.
//
// The decision is module-wide rather than per-file: a migration that leaves one
// file behind — because it names a symbol the replacement dropped — still needs
// the old requirement, while every file it did rewrite needs the new one.
//
// It is also decided from state that does not depend on whether the source
// rewrite has run yet. Every scanner runs before any recipe edits, so in the
// cycle that migrates a file the scan sees it unedited; in a later cycle it sees
// it migrated. Asking "which imports will this file end up with" answers the
// same either way, so the scan runs the source rewrite over the file and reads
// the imports of what it produces.
package depswap

import (
	"strings"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Modules names the migration a scan is following.
type Modules struct {
	// Old is the module being migrated away from.
	Old string
	// New are the modules the migrated source imports. A migration that lands on
	// more than one — an API module and its companion — lists them all.
	New []string
	// Version pins the replacements when a require is introduced or repointed.
	Version string
}

// Acc records which side of the migration the module's source ends up on.
type Acc struct {
	// NeedsOld is set by a file that stays on the old module.
	NeedsOld bool
	// NeedsNew is set by a file that has moved, or that the source rewrite moves.
	NeedsNew bool
}

func NewAcc() *Acc { return &Acc{} }

// Migrate returns what the source rewrite makes of a file, which is how the
// scan predicts where an as-yet-unedited file will land. Recipes supply it as a
// probe over their own editor.
type Migrate func(cu *golang.CompilationUnit) *golang.CompilationUnit

// Scanner records, across every source file, which requirements the module's
// source will need once the migration has run.
func Scanner(acc *Acc, mods Modules, migrate Migrate) recipe.TreeVisitor {
	return visitor.Init(&scanner{acc: acc, mods: mods, migrate: migrate})
}

type scanner struct {
	visitor.GoVisitor
	acc     *Acc
	mods    Modules
	migrate Migrate
}

func (s *scanner) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	onOld, onNew := s.imports(cu)
	// A rewrite can leave the old import behind, for a symbol the replacement
	// dropped, so only the rewritten file's imports tell where it lands.
	if onOld && s.migrate != nil {
		onOld, onNew = s.imports(s.migrate(cu))
	}
	if onOld {
		s.acc.NeedsOld = true
	}
	if onNew {
		s.acc.NeedsNew = true
	}
	return cu
}

func (s *scanner) imports(cu *golang.CompilationUnit) (onOld, onNew bool) {
	if cu.Imports == nil {
		return false, false
	}
	for _, rp := range cu.Imports.Elements {
		path := rp.Element.Path()
		if underModule(path, s.mods.Old) {
			onOld = true
		}
		for _, newModule := range s.mods.New {
			if underModule(path, newModule) {
				onNew = true
			}
		}
	}
	return onOld, onNew
}

func underModule(path, module string) bool {
	return path == module || strings.HasPrefix(path, module+"/")
}

// Editor repoints, adds or leaves the requires according to what the scan found.
func Editor(acc *Acc, mods Modules) recipe.TreeVisitor {
	return visitor.Init(&editor{acc: acc, mods: mods})
}

type editor struct {
	visitor.GoVisitor
	acc  *Acc
	mods Modules
}

func (e *editor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	if !e.acc.NeedsNew {
		return gm
	}
	// Nothing is left on the old module and one takes its place, so repointing
	// it in place keeps the entry's comments and its `// indirect` marker.
	if !e.acc.NeedsOld && len(e.mods.New) == 1 {
		if repointed := migration.ReplaceRequire(gm, e.mods.Old, e.mods.New[0], e.mods.Version); repointed != gm {
			return repointed
		}
	}
	for _, newModule := range e.mods.New {
		gm = migration.AddRequire(gm, newModule, e.mods.Version, false)
	}
	if !e.acc.NeedsOld {
		gm = migration.RemoveRequire(gm, e.mods.Old)
	}
	return gm
}

// EditorProbe runs an editor over a file, for use as a Migrate probe. The
// editor is rebuilt per call, since a recipe's visitor may carry state across
// the files it visits, and its queued after-visits are drained as the recipe
// runner would, since that is where an editor typically adds its imports.
func EditorProbe(newEditor func() recipe.TreeVisitor) Migrate {
	return func(cu *golang.CompilationUnit) *golang.CompilationUnit {
		editor := newEditor()
		ctx := recipe.NewExecutionContext()
		visited := editor.Visit(cu, ctx)
		if visited == nil {
			return cu
		}
		if edited, ok := visitor.DrainAfterVisits(editor, visited, ctx).(*golang.CompilationUnit); ok && edited != nil {
			return edited
		}
		return cu
	}
}
