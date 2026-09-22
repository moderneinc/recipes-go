/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package mapstructurev2

import (
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Repoints mitchellh/mapstructure imports at the go-viper fork.
type SwapMapstructureImports struct {
	recipe.Base
}

func (r *SwapMapstructureImports) Name() string {
	return "org.openrewrite.golang.migration.SwapMapstructureImports"
}
func (r *SwapMapstructureImports) DisplayName() string {
	return "Swap `mitchellh/mapstructure` imports to `go-viper/mapstructure/v2`"
}
func (r *SwapMapstructureImports) Description() string {
	return "Repoint every `github.com/mitchellh/mapstructure` import at `github.com/go-viper/mapstructure/v2`, the maintained fork. The package name is unchanged, so call sites stay as written; only the import path and its type attribution move. A file naming `mapstructure.Error`, the one export v2 dropped, is left alone — see `FindMapstructureErrorUsage`."
}
func (r *SwapMapstructureImports) Tags() []string {
	return []string{"migration", "mapstructure"}
}

func (r *SwapMapstructureImports) Editor() recipe.TreeVisitor {
	return visitor.Init(&swapImportsVisitor{})
}

type swapImportsVisitor struct {
	visitor.GoVisitor
}

func (v *swapImportsVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	if !pathswap.Imports(cu, oldModule) {
		return cu
	}
	// Swapping while the file still binds `mapstructure` to the fork would
	// duplicate the import, and the two packages share a name.
	if pathswap.Imports(cu, newModule) {
		return cu
	}
	if referencesRemovedError(cu) {
		return cu
	}

	swapped := pathswap.RewriteImports(cu, swapRules)
	if swapped == cu {
		return cu
	}
	// The local name survives the swap, so every reference still carries
	// parse-time attribution naming the old module; RemoveUnusedImports would
	// then read the new import as unused and drop it.
	if retyped, ok := pathswap.Retype(swapRules).Visit(swapped, p).(*golang.CompilationUnit); ok {
		return retyped
	}
	return swapped
}
