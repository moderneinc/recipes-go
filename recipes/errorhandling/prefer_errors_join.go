/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"fmt"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var ejErr = template.Expr("ejErr")

var errorWrapPattern = template.Expression(fmt.Sprintf(`fmt.Errorf("%%w", %s)`, ejErr)).
	Captures(ejErr).Imports("fmt").Build()

// Replaces `fmt.Errorf("%w", err)` with `err` when wrapping adds no context.
type SimplifyRedundantErrorWrap struct {
	recipe.Base
}

func (r *SimplifyRedundantErrorWrap) Name() string {
	return "org.openrewrite.golang.codequality.SimplifyRedundantErrorWrap"
}
func (r *SimplifyRedundantErrorWrap) DisplayName() string { return "Simplify redundant error wrap" }
func (r *SimplifyRedundantErrorWrap) Description() string {
	return "Replace `fmt.Errorf(\"%w\", err)` with `err` when wrapping adds no context."
}
func (r *SimplifyRedundantErrorWrap) Tags() []string { return []string{"error-handling"} }

func (r *SimplifyRedundantErrorWrap) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifyRedundantErrorWrapVisitor{})
}

type simplifyRedundantErrorWrapVisitor struct {
	visitor.GoVisitor
}

func (v *simplifyRedundantErrorWrapVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	match := errorWrapPattern.Match(mi, nil)
	if match == nil {
		return mi
	}

	// Skip unless the wrapped value is an error, since replacing fmt.Errorf (which
	// returns error) with the bare value would otherwise not compile.
	arg, ok := match.GetCapture(ejErr).(java.Expression)
	if !ok || !isErrorAssignable(arg) {
		return mi
	}

	recipegolang.MaybeRemoveImport(v, "fmt")
	return setExprPrefixLocal(stripExprPrefix(arg), mi.GetPrefix())
}
