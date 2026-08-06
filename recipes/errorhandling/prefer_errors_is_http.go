/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Replaces `err == http.ErrServerClosed` with
// `errors.Is(err, http.ErrServerClosed)` (and the negated form) for correct
// wrapped error handling.
type PreferErrorsIsHttpServerClosed struct {
	recipe.Base
}

func (r *PreferErrorsIsHttpServerClosed) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsHttpServerClosed"
}
func (r *PreferErrorsIsHttpServerClosed) DisplayName() string {
	return "Prefer errors.Is for http.ErrServerClosed comparison"
}
func (r *PreferErrorsIsHttpServerClosed) Description() string {
	return "Replace `err == http.ErrServerClosed` with `errors.Is(err, http.ErrServerClosed)` and `err != http.ErrServerClosed` with `!errors.Is(err, http.ErrServerClosed)` for correct wrapped error handling."
}
func (r *PreferErrorsIsHttpServerClosed) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsHttpServerClosed) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferErrorsIsHttpVisitor{})
}

type preferErrorsIsHttpVisitor struct {
	visitor.GoVisitor
}

func (v *preferErrorsIsHttpVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)

	if bin.Operator.Element != java.Equal && bin.Operator.Element != java.NotEqual {
		return bin
	}
	if errExpr, sentinel, ok := matchSentinel(bin, "http", "ErrServerClosed"); ok {
		return rewriteToErrorsIs(v, bin, errExpr, sentinel)
	}
	return bin
}
