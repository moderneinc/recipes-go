/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"fmt"

	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var sprintfArg = template.Expr("s")

var redundantSprintfPattern = template.Expression(fmt.Sprintf(`fmt.Sprintf("%%s", %s)`, sprintfArg)).
	Captures(sprintfArg).Imports("fmt").Build()

// Replaces `fmt.Sprintf("%s", s)` with `s` when the format string is a single
// %s and the argument is a string.
// Staticcheck: S1025
type RemoveRedundantSprintf struct {
	recipe.Base
}

func (r *RemoveRedundantSprintf) Name() string {
	return "org.openrewrite.golang.codequality.RemoveRedundantSprintf"
}
func (r *RemoveRedundantSprintf) DisplayName() string { return "Remove redundant fmt.Sprintf" }
func (r *RemoveRedundantSprintf) Description() string {
	return "Replace `fmt.Sprintf(\"%s\", s)` with `s` when the format string is a single %s."
}
func (r *RemoveRedundantSprintf) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *RemoveRedundantSprintf) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1025", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *RemoveRedundantSprintf) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeRedundantSprintfVisitor{})
}

type removeRedundantSprintfVisitor struct {
	visitor.GoVisitor
}

func (v *removeRedundantSprintfVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	match := redundantSprintfPattern.Match(mi, nil)
	if match == nil {
		return mi
	}

	// Skip unless the argument is a string, since %s also formats []byte, a
	// fmt.Stringer, or a named string type, none of which are a plain string.
	arg, ok := match.GetCapture(sprintfArg).(java.Expression)
	if !ok || !matcher.IsString(matcher.TypeOfExpression(arg)) {
		return mi
	}

	recipegolang.MaybeRemoveImport(v, "fmt")
	return withLeadingPrefix(arg, mi.GetPrefix())
}

// Returns e with its leading prefix set to p, covering the expression kinds that
// appear as a fmt.Sprintf string argument.
func withLeadingPrefix(e java.Expression, p java.Space) java.Expression {
	switch n := e.(type) {
	case *java.Identifier:
		return n.WithPrefix(p)
	case *java.Literal:
		return n.WithPrefix(p)
	case *java.FieldAccess:
		return n.WithPrefix(p)
	case *java.MethodInvocation:
		return n.WithPrefix(p)
	case *java.Parentheses:
		return n.WithPrefix(p)
	case *java.Binary:
		return n.WithPrefix(p)
	case *java.ArrayAccess:
		return n.WithPrefix(p)
	case *golang.TypeAssertion:
		return n.WithPrefix(p)
	}
	return e
}
