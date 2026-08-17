/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Replaces `err == sql.ErrNoRows` with `errors.Is(err, sql.ErrNoRows)` (and the
// negated form) for correct wrapped error handling.
type PreferErrorsIsSqlNoRows struct {
	recipe.Base
}

func (r *PreferErrorsIsSqlNoRows) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsSqlNoRows"
}
func (r *PreferErrorsIsSqlNoRows) DisplayName() string {
	return "Prefer errors.Is for sql.ErrNoRows comparison"
}
func (r *PreferErrorsIsSqlNoRows) Description() string {
	return "Replace `err == sql.ErrNoRows` with `errors.Is(err, sql.ErrNoRows)` and `err != sql.ErrNoRows` with `!errors.Is(err, sql.ErrNoRows)` for correct wrapped error handling."
}
func (r *PreferErrorsIsSqlNoRows) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsSqlNoRows) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "errorlint", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *PreferErrorsIsSqlNoRows) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferErrorsIsSqlVisitor{})
}

type preferErrorsIsSqlVisitor struct {
	visitor.GoVisitor
}

func (v *preferErrorsIsSqlVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)

	if bin.Operator.Element != java.Equal && bin.Operator.Element != java.NotEqual {
		return bin
	}
	if errExpr, sentinel, ok := matchSentinel(bin, "sql", "ErrNoRows"); ok {
		return rewriteToErrorsIs(v, bin, errExpr, sentinel)
	}
	return bin
}
