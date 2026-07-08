/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package migration

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
)

// GoModTidy applies `go mod tidy` behavior through composed recipes: it adds
// missing requirements (when the module graph was resolved at parse time),
// corrects the `// indirect` markers on requirements, and canonicalizes the
// ordering of require blocks.
//
// AddMissingGoModRequires needs the parse-time toolchain resolution the
// rewrite-go parser performs; without it that step is a no-op and the composite
// still applies the offline-safe indirect-marker and formatting fixes. It does
// not remove unused requirements or sync go.sum; use FindUnusedGoModRequires to
// report removal candidates and run `go mod tidy` for a go.sum sync.
type GoModTidy struct {
	recipe.Base
}

func (r *GoModTidy) Name() string { return "org.openrewrite.golang.migration.GoModTidy" }

func (r *GoModTidy) DisplayName() string { return "Tidy go.mod" }

func (r *GoModTidy) Description() string {
	return "Apply `go mod tidy` behavior: add missing requirements at their resolved versions (when the module graph was resolved at parse time), correct the `// indirect` markers on requirements, and sort require blocks. " +
		"It does not remove unused requirements or sync go.sum; use `FindUnusedGoModRequires` to report removal candidates and run `go mod tidy` for a go.sum sync."
}

func (r *GoModTidy) Tags() []string { return []string{"gomod", "tidy"} }

func (r *GoModTidy) RecipeList() []recipe.Recipe {
	return []recipe.Recipe{
		&AddMissingGoModRequires{},
		&FixGoModIndirectMarkers{},
		&FormatGoMod{},
	}
}
