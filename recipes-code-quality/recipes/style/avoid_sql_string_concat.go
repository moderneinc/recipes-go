/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// sqlKeywords lists SQL keywords that indicate a query string.
var sqlKeywords = []string{"SELECT", "INSERT", "UPDATE", "DELETE", "DROP"}

// AvoidSqlStringConcat finds string concatenation where the left operand is a
// string literal containing SQL keywords (SELECT, INSERT, UPDATE, DELETE, DROP).
// Building SQL queries via string concatenation is a common source of SQL
// injection vulnerabilities. Use parameterized queries instead.
type AvoidSqlStringConcat struct {
	recipe.Base
}

func (r *AvoidSqlStringConcat) Name() string {
	return "org.openrewrite.golang.codequality.AvoidSqlStringConcat"
}
func (r *AvoidSqlStringConcat) DisplayName() string { return "Avoid SQL string concatenation" }
func (r *AvoidSqlStringConcat) Description() string {
	return "Find string concatenation where the left operand contains SQL keywords. Use parameterized queries to avoid SQL injection."
}
func (r *AvoidSqlStringConcat) Tags() []string { return []string{"security"} }

func (r *AvoidSqlStringConcat) Editor() recipe.TreeVisitor {
	return visitor.Init(&avoidSqlStringConcatVisitor{})
}

type avoidSqlStringConcatVisitor struct {
	visitor.GoVisitor
}

func (v *avoidSqlStringConcatVisitor) VisitBinary(bin *tree.Binary, p any) tree.J {
	bin = v.GoVisitor.VisitBinary(bin, p).(*tree.Binary)

	if bin.Operator.Element != tree.Add {
		return bin
	}

	lit, ok := bin.Left.(*tree.Literal)
	if !ok || lit.Kind != tree.StringLiteral {
		return bin
	}

	upper := strings.ToUpper(lit.Source)
	for _, kw := range sqlKeywords {
		if strings.Contains(upper, kw) {
			bin = bin.WithMarkers(tree.MarkupWarn(bin.Markers, "possible SQL injection via string concatenation"))
			return bin
		}
	}

	return bin
}
