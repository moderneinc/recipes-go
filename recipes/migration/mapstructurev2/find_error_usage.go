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

// Reports the references that block the migration to v2.
type FindMapstructureErrorUsage struct {
	recipe.Base
}

func (r *FindMapstructureErrorUsage) Name() string {
	return "org.openrewrite.golang.migration.FindMapstructureErrorUsage"
}
func (r *FindMapstructureErrorUsage) DisplayName() string {
	return "Find `mapstructure.Error` usage"
}
func (r *FindMapstructureErrorUsage) Description() string {
	return "Mark every reference to `mapstructure.Error`, the one `github.com/mitchellh/mapstructure` export that `github.com/go-viper/mapstructure/v2` dropped. v2 joins its decode failures with `errors.Join` and exposes `Error` as an interface, so code reading the struct's `Errors []string` field has to be reworked by hand before the module can move."
}
func (r *FindMapstructureErrorUsage) Tags() []string {
	return []string{"search", "migration", "mapstructure"}
}

func (r *FindMapstructureErrorUsage) Editor() recipe.TreeVisitor {
	return visitor.Init(&findErrorUsageVisitor{})
}

type findErrorUsageVisitor struct {
	visitor.GoVisitor
	pkg string
}

func (v *findErrorUsageVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.pkg = pathswap.Qualifier(cu, oldModule)
	if v.pkg == "" {
		return cu
	}
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *findErrorUsageVisitor) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)
	if !isRemovedErrorRef(fa, v.pkg) {
		return fa
	}
	return fa.WithMarkers(java.MarkupWarn(fa.Markers,
		"mapstructure.Error is not in go-viper/mapstructure/v2; the fork joins decode failures with errors.Join, so unwrap them with errors.Is/errors.As instead of reading Errors []string"))
}
