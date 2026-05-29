/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveRedundantReturn removes bare `return` statements at the end of
// functions that have no return values.
// Staticcheck: S1023
type RemoveRedundantReturn struct {
	recipe.Base
}

func (r *RemoveRedundantReturn) Name() string {
	return "org.openrewrite.golang.codequality.RemoveRedundantReturn"
}
func (r *RemoveRedundantReturn) DisplayName() string { return "Remove redundant return" }
func (r *RemoveRedundantReturn) Description() string {
	return "Remove bare `return` at the end of functions with no return values."
}
func (r *RemoveRedundantReturn) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *RemoveRedundantReturn) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1023", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *RemoveRedundantReturn) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeRedundantReturnVisitor{})
}

type removeRedundantReturnVisitor struct {
	visitor.GoVisitor
}

func (v *removeRedundantReturnVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)

	// Only apply to functions with no return type.
	if md.ReturnType != nil || md.Body == nil {
		return md
	}

	stmts := md.Body.Statements
	if len(stmts) == 0 {
		return md
	}

	// Check if the last statement is a bare return (no return values).
	last := stmts[len(stmts)-1]
	ret, ok := last.Element.(*java.Return)
	if !ok {
		return md
	}
	if len(ret.Expressions) > 0 {
		return md
	}

	// Remove the trailing bare return.
	md = md.WithBody(md.Body.WithStatements(stmts[:len(stmts)-1]))
	return md
}
