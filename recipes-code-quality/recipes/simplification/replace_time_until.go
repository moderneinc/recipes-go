/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var untilTimeNowMatcher = matcher.NewMethodMatcher("time Now()")

// ReplaceTimeUntilWithUntil replaces `t.Sub(time.Now())` with `time.Until(t)`.
// Staticcheck: S1024
type ReplaceTimeUntilWithUntil struct {
	recipe.Base
}

func (r *ReplaceTimeUntilWithUntil) Name() string {
	return "org.openrewrite.golang.codequality.ReplaceTimeUntilWithUntil"
}
func (r *ReplaceTimeUntilWithUntil) DisplayName() string {
	return "Replace t.Sub(time.Now()) with time.Until(t)"
}
func (r *ReplaceTimeUntilWithUntil) Description() string {
	return "Replace `t.Sub(time.Now())` with `time.Until(t)` for clarity."
}
func (r *ReplaceTimeUntilWithUntil) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *ReplaceTimeUntilWithUntil) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1024", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *ReplaceTimeUntilWithUntil) Editor() recipe.TreeVisitor {
	return visitor.Init(&replaceTimeUntilVisitor{})
}

type replaceTimeUntilVisitor struct {
	visitor.GoVisitor
}

func (v *replaceTimeUntilVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	// Match: <receiver>.Sub(arg)
	if mi.Name.Name != "Sub" || mi.Select == nil {
		return mi
	}

	// The argument must be time.Now()
	args := mi.Arguments.Elements
	var argExpr java.Expression
	for _, a := range args {
		if _, isEmpty := a.Element.(*java.Empty); !isEmpty {
			if argExpr != nil {
				return mi // more than one real arg
			}
			argExpr = a.Element
		}
	}
	if argExpr == nil {
		return mi
	}

	nowCall, ok := argExpr.(*java.MethodInvocation)
	if !ok || !untilTimeNowMatcher.Matches(nowCall) {
		return mi
	}

	// The receiver (mi.Select) is the time value `t` to pass to time.Until(t)
	receiver := mi.Select.Element

	// Get the leading prefix from the receiver for the replacement
	prefix := exprPrefixOf(receiver)

	newTimeIdent := &java.Identifier{Prefix: prefix, Name: "time"}
	untilIdent := &java.Identifier{Name: "Until"}

	// Build argument list with the receiver as the arg
	receiverWithNoPrefix := setExprPrefixOf(receiver, java.Space{})
	newArgs := java.Container[java.Expression]{
		Before:   mi.Arguments.Before,
		Elements: []java.RightPadded[java.Expression]{{Element: receiverWithNoPrefix}},
	}

	return &java.MethodInvocation{
		Select:    &java.RightPadded[java.Expression]{Element: newTimeIdent, After: mi.Select.After},
		Name:      untilIdent,
		Arguments: newArgs,
	}
}

func exprPrefixOf(expr java.Expression) java.Space {
	switch n := expr.(type) {
	case *java.Identifier:
		return n.Prefix
	case *java.FieldAccess:
		return exprPrefixOf(n.Target)
	case *java.MethodInvocation:
		if n.Select != nil {
			return exprPrefixOf(n.Select.Element)
		}
		return exprPrefixOf(n.Name)
	default:
		return java.Space{}
	}
}

func setExprPrefixOf(expr java.Expression, prefix java.Space) java.Expression {
	switch n := expr.(type) {
	case *java.Identifier:
		return n.WithPrefix(prefix)
	case *java.FieldAccess:
		return n.WithPrefix(prefix)
	case *java.MethodInvocation:
		return n.WithPrefix(prefix)
	default:
		return expr
	}
}
