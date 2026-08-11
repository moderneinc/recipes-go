/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Replaces `err == context.Canceled` and `err == context.DeadlineExceeded` with
// `errors.Is` (and the negated forms) for correct wrapped error handling.
type PreferErrorsIsContext struct {
	recipe.Base
}

func (r *PreferErrorsIsContext) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsContext"
}
func (r *PreferErrorsIsContext) DisplayName() string {
	return "Prefer errors.Is for context error comparison"
}
func (r *PreferErrorsIsContext) Description() string {
	return "Replace `err == context.Canceled` and `err == context.DeadlineExceeded` with `errors.Is` for correct wrapped error handling."
}
func (r *PreferErrorsIsContext) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsContext) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferErrorsIsContextVisitor{})
}

type preferErrorsIsContextVisitor struct {
	visitor.GoVisitor
}

func (v *preferErrorsIsContextVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)

	if bin.Operator.Element != java.Equal && bin.Operator.Element != java.NotEqual {
		return bin
	}
	for _, name := range []string{"Canceled", "DeadlineExceeded"} {
		if errExpr, sentinel, ok := matchSentinel(bin, "context", name); ok {
			return rewriteToErrorsIs(v, bin, errExpr, sentinel)
		}
	}
	return bin
}
