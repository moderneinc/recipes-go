/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// sqlKeywords lists SQL keywords that indicate a query string.
var sqlKeywords = []string{"SELECT", "INSERT", "UPDATE", "DELETE", "DROP"}

// FindSQLStringConcat finds string concatenation where the left operand is a
// string literal containing SQL keywords (SELECT, INSERT, UPDATE, DELETE, DROP).
// Building SQL queries via string concatenation is a common source of SQL
// injection vulnerabilities. Use parameterized queries instead.
type FindSQLStringConcat struct {
	recipe.Base
}

func (r *FindSQLStringConcat) Name() string {
	return "org.openrewrite.golang.codequality.FindSQLStringConcat"
}
func (r *FindSQLStringConcat) DisplayName() string { return "Find SQL string concatenation" }
func (r *FindSQLStringConcat) Description() string {
	return "Find string concatenation where the left operand contains SQL keywords. Use parameterized queries to avoid SQL injection."
}
func (r *FindSQLStringConcat) Tags() []string { return []string{"security"} }

func (r *FindSQLStringConcat) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSQLStringConcatVisitor{})
}

type findSQLStringConcatVisitor struct {
	visitor.GoVisitor
}

func (v *findSQLStringConcatVisitor) VisitBinary(bin *tree.Binary, p any) tree.J {
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
			bin = bin.WithMarkers(tree.FoundSearchResult(bin.Markers, "possible SQL injection via string concatenation"))
			return bin
		}
	}

	return bin
}
