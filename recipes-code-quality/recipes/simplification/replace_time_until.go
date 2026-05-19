/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
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

func (v *replaceTimeUntilVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Match: <receiver>.Sub(arg)
	if mi.Name.Name != "Sub" || mi.Select == nil {
		return mi
	}

	// The argument must be time.Now()
	args := mi.Arguments.Elements
	var argExpr tree.Expression
	for _, a := range args {
		if _, isEmpty := a.Element.(*tree.Empty); !isEmpty {
			if argExpr != nil {
				return mi // more than one real arg
			}
			argExpr = a.Element
		}
	}
	if argExpr == nil {
		return mi
	}

	nowCall, ok := argExpr.(*tree.MethodInvocation)
	if !ok || !untilTimeNowMatcher.Matches(nowCall) {
		return mi
	}

	// The receiver (mi.Select) is the time value `t` to pass to time.Until(t)
	receiver := mi.Select.Element

	// Get the leading prefix from the receiver for the replacement
	prefix := exprPrefixOf(receiver)

	newTimeIdent := &tree.Identifier{Prefix: prefix, Name: "time"}
	untilIdent := &tree.Identifier{Name: "Until"}

	// Build argument list with the receiver as the arg
	receiverWithNoPrefix := setExprPrefixOf(receiver, tree.Space{})
	newArgs := tree.Container[tree.Expression]{
		Before:   mi.Arguments.Before,
		Elements: []tree.RightPadded[tree.Expression]{{Element: receiverWithNoPrefix}},
	}

	return &tree.MethodInvocation{
		Select:    &tree.RightPadded[tree.Expression]{Element: newTimeIdent, After: mi.Select.After},
		Name:      untilIdent,
		Arguments: newArgs,
	}
}

func exprPrefixOf(expr tree.Expression) tree.Space {
	switch n := expr.(type) {
	case *tree.Identifier:
		return n.Prefix
	case *tree.FieldAccess:
		return exprPrefixOf(n.Target)
	case *tree.MethodInvocation:
		if n.Select != nil {
			return exprPrefixOf(n.Select.Element)
		}
		return exprPrefixOf(n.Name)
	default:
		return tree.Space{}
	}
}

func setExprPrefixOf(expr tree.Expression, prefix tree.Space) tree.Expression {
	switch n := expr.(type) {
	case *tree.Identifier:
		return n.WithPrefix(prefix)
	case *tree.FieldAccess:
		return n.WithPrefix(prefix)
	case *tree.MethodInvocation:
		return n.WithPrefix(prefix)
	default:
		return expr
	}
}
