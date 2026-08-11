/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Replaces `err == net.ErrClosed` with `errors.Is(err, net.ErrClosed)` (and the
// negated form) for correct wrapped error handling.
type PreferErrorsIsNetClosed struct {
	recipe.Base
}

func (r *PreferErrorsIsNetClosed) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsNetClosed"
}
func (r *PreferErrorsIsNetClosed) DisplayName() string {
	return "Prefer errors.Is for net.ErrClosed comparison"
}
func (r *PreferErrorsIsNetClosed) Description() string {
	return "Replace `err == net.ErrClosed` with `errors.Is(err, net.ErrClosed)` and `err != net.ErrClosed` with `!errors.Is(err, net.ErrClosed)` for correct wrapped error handling."
}
func (r *PreferErrorsIsNetClosed) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsNetClosed) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferErrorsIsNetVisitor{})
}

type preferErrorsIsNetVisitor struct {
	visitor.GoVisitor
}

func (v *preferErrorsIsNetVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)

	if bin.Operator.Element != java.Equal && bin.Operator.Element != java.NotEqual {
		return bin
	}
	if errExpr, sentinel, ok := matchSentinel(bin, "net", "ErrClosed"); ok {
		return rewriteToErrorsIs(v, bin, errExpr, sentinel)
	}
	return bin
}
