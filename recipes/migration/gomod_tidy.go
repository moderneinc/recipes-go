/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// GoModTidy applies the subset of `go mod tidy` that is safe to run offline: it
// corrects the `// indirect` markers on requirements and canonicalizes the
// ordering of require blocks.
//
// It deliberately does not add missing requirements, remove unused ones, or
// sync go.sum: those need to resolve module versions and the transitive module
// graph over the network. Use FindMissingGoModRequires and
// FindUnusedGoModRequires to report what a full `go mod tidy` would add or
// remove, and run `go mod tidy` itself for a complete sync.
type GoModTidy struct {
	recipe.Base
}

func (r *GoModTidy) Name() string { return "org.openrewrite.golang.migration.GoModTidy" }

func (r *GoModTidy) DisplayName() string { return "Tidy go.mod (offline)" }

func (r *GoModTidy) Description() string {
	return "Apply the offline-safe subset of `go mod tidy`: correct the `// indirect` markers on requirements and sort require blocks. " +
		"It does not add missing requirements, remove unused ones, or sync go.sum, since those require resolving module versions and the transitive module graph over the network; run `go mod tidy` for a full sync."
}

func (r *GoModTidy) Tags() []string { return []string{"gomod", "tidy"} }

func (r *GoModTidy) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		&FixGoModIndirectMarkers{},
		&FormatGoMod{},
	}
}
