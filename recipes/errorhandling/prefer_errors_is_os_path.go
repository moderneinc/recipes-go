/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Replaces `err == os.ErrInvalid` with `errors.Is(err, os.ErrInvalid)` for
// correct wrapped error handling.
type PreferErrorsIsOsInvalid struct {
	recipe.Base
}

func (r *PreferErrorsIsOsInvalid) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorsIsOsInvalid"
}
func (r *PreferErrorsIsOsInvalid) DisplayName() string {
	return "Prefer errors.Is for os.ErrInvalid comparison"
}
func (r *PreferErrorsIsOsInvalid) Description() string {
	return "Replace `err == os.ErrInvalid` with `errors.Is(err, os.ErrInvalid)` for correct wrapped error handling."
}
func (r *PreferErrorsIsOsInvalid) Tags() []string { return []string{"error-handling"} }

func (r *PreferErrorsIsOsInvalid) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferErrorsIsOsVisitor{})
}

type preferErrorsIsOsVisitor struct {
	visitor.GoVisitor
}

func (v *preferErrorsIsOsVisitor) VisitBinary(bin *java.Binary, p any) java.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*java.Binary)

	if bin.Operator.Element != java.Equal && bin.Operator.Element != java.NotEqual {
		return bin
	}
	if errExpr, sentinel, ok := matchSentinel(bin, "os", "ErrInvalid"); ok {
		return rewriteToErrorsIs(v, bin, errExpr, sentinel)
	}
	return bin
}
