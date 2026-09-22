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
// rewrite has run yet. The recipe runner in rewrite-go's test harness walks a
// recipe list in order over one shared set of trees, so a later scanning recipe
// sees the edits; the Moderne CLI scans before the list's edits are applied and
// would see none of them. Asking "which side of the migration will this file end
// up on" answers the same either way: a file already on the new module counts as
// migrated, and so does one on the old module that the source rewrite would
// take.
package depswap

import (
	"strings"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
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

// Migrates reports whether the source rewrite would take a file, which is how
// the scan predicts where an as-yet-unedited file will land. Recipes supply it
// as a probe over their own editor.
type Migrates func(cu *golang.CompilationUnit) bool

// Scanner records, across every source file, which requirements the module's
// source will need once the migration has run.
func Scanner(acc *Acc, mods Modules, migrates Migrates) recipe.TreeVisitor {
	return visitor.Init(&scanner{acc: acc, mods: mods, migrates: migrates})
}

type scanner struct {
	visitor.GoVisitor
	acc      *Acc
	mods     Modules
	migrates Migrates
}

func (s *scanner) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	if cu.Imports == nil {
		return cu
	}
	onOld, onNew := false, false
	for _, rp := range cu.Imports.Elements {
		path := pathswap.Path(rp.Element)
		if underModule(path, s.mods.Old) {
			onOld = true
		}
		for _, newModule := range s.mods.New {
			if underModule(path, newModule) {
				onNew = true
			}
		}
	}

	if onNew {
		s.acc.NeedsNew = true
	}
	if onOld {
		// A file the rewrite takes will import the new module; one it refuses
		// keeps the old requirement alive.
		if s.migrates != nil && s.migrates(cu) {
			s.acc.NeedsNew = true
		} else {
			s.acc.NeedsOld = true
		}
	}
	return cu
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

// EditorProbe reports whether an editor changes a file, for use as a Migrates
// predicate. The editor is rebuilt per call, since a recipe's visitor may carry
// state across the files it visits.
func EditorProbe(newEditor func() recipe.TreeVisitor) Migrates {
	return func(cu *golang.CompilationUnit) bool {
		return newEditor().Visit(cu, recipe.NewExecutionContext()) != java.Tree(cu)
	}
}
