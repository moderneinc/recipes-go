/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// GoModTidy applies `go mod tidy` behavior to go.mod through composed recipes:
// it adds missing requirements, removes unused ones, corrects the `// indirect`
// markers, and canonicalizes the ordering of require blocks.
//
// Adding and removing requirements need the module graph the rewrite-go parser
// resolves at parse time; without it those steps are no-ops and the composite
// still applies the offline-safe indirect-marker and formatting fixes. It does
// not sync go.sum; the upstream `RegenerateGoSum` recipe covers that.
type GoModTidy struct {
	recipe.Base
}

func (r *GoModTidy) Name() string { return "org.openrewrite.golang.migration.GoModTidy" }

func (r *GoModTidy) DisplayName() string { return "Tidy go.mod" }

func (r *GoModTidy) Description() string {
	return "Apply `go mod tidy` behavior to go.mod: add missing requirements at their resolved versions, remove unused ones, correct the `// indirect` markers, and sort require blocks. " +
		"Adding and removing require the module graph resolved at parse time, and are no-ops without it. It does not sync go.sum; the `RegenerateGoSum` recipe covers that."
}

func (r *GoModTidy) Tags() []string { return []string{"gomod", "tidy"} }

func (r *GoModTidy) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		&AddMissingGoModRequires{},
		&RemoveUnusedGoModRequires{},
		&FixGoModIndirectMarkers{},
		&FormatGoMod{},
	}
}
