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

// Replaces `err == io.EOF` with `errors.Is(err, io.EOF)` (and the negated form)
// for correct wrapped error handling.
type PreferErrorsIsEOF struct {
	recipe.Base
}

func (r *PreferErrorsIsEOF) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsEOF"
}
func (r *PreferErrorsIsEOF) DisplayName() string {
	return "Prefer errors.Is for io.EOF comparison"
}
func (r *PreferErrorsIsEOF) Description() string {
	return "Replace `err == io.EOF` with `errors.Is(err, io.EOF)` and `err != io.EOF` with `!errors.Is(err, io.EOF)` for correct wrapped error handling."
}
func (r *PreferErrorsIsEOF) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsEOF) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "errorlint", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *PreferErrorsIsEOF) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferErrorsIsEOFVisitor{})
}

type preferErrorsIsEOFVisitor struct {
	visitor.GoVisitor
}

func (v *preferErrorsIsEOFVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)

	if bin.Operator.Element != java.Equal && bin.Operator.Element != java.NotEqual {
		return bin
	}
	if errExpr, sentinel, ok := matchSentinel(bin, "io", "EOF"); ok {
		return rewriteToErrorsIs(v, bin, errExpr, sentinel)
	}
	return bin
}
