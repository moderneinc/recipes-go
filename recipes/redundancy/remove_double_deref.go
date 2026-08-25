/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"fmt"

	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
)

// RemoveDoubleDeref removes `*&x` patterns where taking the address of a variable
// and immediately dereferencing it is a no-op. The expression `*&x` is
// replaced with just `x`.
type RemoveDoubleDeref struct {
	recipe.Base
}

func (r *RemoveDoubleDeref) Name() string {
	return "org.openrewrite.golang.codequality.RemoveDoubleDeref"
}
func (r *RemoveDoubleDeref) DisplayName() string { return "Remove redundant *& (deref of address-of)" }
func (r *RemoveDoubleDeref) Description() string {
	return "Remove `*&x` where taking the address and immediately dereferencing is a no-op."
}
func (r *RemoveDoubleDeref) Tags() []string { return []string{"cleanup", "redundancy"} }

func (r *RemoveDoubleDeref) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA4001", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *RemoveDoubleDeref) Editor() recipe.TreeVisitor {
	return template.Rewrite(doubleDerefPattern, doubleDerefTemplate)
}

var (
	doubleDerefOperand  = template.Expr("x")
	doubleDerefPattern  = template.Expression(fmt.Sprintf("*&%s", doubleDerefOperand)).Captures(doubleDerefOperand).Build()
	doubleDerefTemplate = template.ExpressionTemplate(doubleDerefOperand.String()).Captures(doubleDerefOperand).Build()
)
