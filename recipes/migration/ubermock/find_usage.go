/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package ubermock

import (
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Reports every import of the archived mock library.
type FindGolangMockUsage struct {
	recipe.Base
}

func (r *FindGolangMockUsage) Name() string {
	return "org.openrewrite.golang.migration.FindGolangMockUsage"
}
func (r *FindGolangMockUsage) DisplayName() string {
	return "Find `github.com/golang/mock` usage"
}
func (r *FindGolangMockUsage) Description() string {
	return "Mark every import of `github.com/golang/mock`, archived by Google in June 2023, with the `go.uber.org/mock` path that replaces it."
}
func (r *FindGolangMockUsage) Tags() []string {
	return []string{"search", "migration", "mock", "testing"}
}

func (r *FindGolangMockUsage) Editor() recipe.TreeVisitor {
	return visitor.Init(&findUsageVisitor{})
}

type findUsageVisitor struct {
	visitor.GoVisitor
}

func (v *findUsageVisitor) VisitImport(imp *java.Import, p any) java.J {
	imp = v.GoVisitor.VisitImport(imp, p).(*java.Import)
	newPath, ok := pathswap.MapPath(pathswap.Path(imp), swapRules)
	if !ok {
		return imp
	}
	return imp.WithMarkers(java.MarkupInfo(imp.Markers,
		"github.com/golang/mock is archived; migrate this import to "+newPath))
}
