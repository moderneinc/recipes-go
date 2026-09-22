/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package ubermock

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Repoints every github.com/golang/mock import at go.uber.org/mock.
type SwapGolangMockImports struct {
	recipe.Base
}

func (r *SwapGolangMockImports) Name() string {
	return "org.openrewrite.golang.migration.SwapGolangMockImports"
}
func (r *SwapGolangMockImports) DisplayName() string {
	return "Swap `github.com/golang/mock` imports to `go.uber.org/mock`"
}
func (r *SwapGolangMockImports) Description() string {
	return "Repoint every `github.com/golang/mock` import at `go.uber.org/mock`, which covers `gomock`, `mockgen` and `mockgen/model`. The fork's API is a superset of the original's, so call sites are unchanged; only the import path and its type attribution move."
}
func (r *SwapGolangMockImports) Tags() []string { return []string{"migration", "mock", "testing"} }

func (r *SwapGolangMockImports) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *SwapGolangMockImports) Editor() recipe.TreeVisitor {
	return visitor.Init(&swapImportsVisitor{})
}

type swapImportsVisitor struct {
	visitor.GoVisitor
}

func (v *swapImportsVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	if !importsOldModule(cu) || importsBothForks(cu) {
		return cu
	}
	swapped := pathswap.RewriteImports(cu, swapRules)
	if swapped == cu {
		return cu
	}
	// The local name is unchanged by the swap, so every reference still carries
	// parse-time attribution naming github.com/golang/mock. Left alone,
	// RemoveUnusedImports reads the tree as still using the old package and drops
	// the import this recipe just introduced.
	if retyped, ok := pathswap.Retype(swapRules).Visit(swapped, p).(*golang.CompilationUnit); ok {
		return retyped
	}
	return swapped
}

// importsOldModule reports whether cu imports anything under
// github.com/golang/mock.
func importsOldModule(cu *golang.CompilationUnit) bool {
	if cu == nil || cu.Imports == nil {
		return false
	}
	for _, rp := range cu.Imports.Elements {
		if _, ok := pathswap.MapPath(pathswap.Path(rp.Element), swapRules); ok {
			return true
		}
	}
	return false
}

// importsBothForks reports whether cu imports a path that the swap would
// duplicate, because the corresponding go.uber.org/mock path is already
// imported. Such a file is left for review rather than made uncompilable.
func importsBothForks(cu *golang.CompilationUnit) bool {
	if cu == nil || cu.Imports == nil {
		return false
	}
	existing := map[string]bool{}
	for _, rp := range cu.Imports.Elements {
		existing[pathswap.Path(rp.Element)] = true
	}
	for _, rp := range cu.Imports.Elements {
		if newPath, ok := pathswap.MapPath(pathswap.Path(rp.Element), swapRules); ok && existing[newPath] {
			return true
		}
	}
	return false
}
