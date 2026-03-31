/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// sqlQueryMethods lists database methods that accept SQL strings.
var sqlQueryMethods = map[string]bool{
	"Query":    true,
	"QueryRow": true,
	"Exec":     true,
}

// FindSqlQueryConcat finds SQL queries built with string concatenation, e.g.
// `db.Query("SELECT * FROM " + table)`. Building SQL via concatenation is a
// common source of SQL injection vulnerabilities. Use parameterized queries
// instead.
type FindSqlQueryConcat struct {
	recipe.Base
}

func (r *FindSqlQueryConcat) Name() string {
	return "org.openrewrite.golang.codequality.FindSqlQueryConcat"
}
func (r *FindSqlQueryConcat) DisplayName() string { return "Find SQL query string concatenation" }
func (r *FindSqlQueryConcat) Description() string {
	return "Find SQL queries built with string concatenation via db.Query, db.QueryRow, or db.Exec. Use parameterized queries to avoid SQL injection."
}
func (r *FindSqlQueryConcat) Tags() []string { return []string{"security", "database/sql"} }

func (r *FindSqlQueryConcat) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSqlQueryConcatVisitor{})
}

type findSqlQueryConcatVisitor struct {
	visitor.GoVisitor
}

func (v *findSqlQueryConcatVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	if !sqlQueryMethods[mi.Name.Name] {
		return mi
	}

	// Must have at least one argument (the query string)
	if len(mi.Arguments.Elements) == 0 {
		return mi
	}

	// Check if the first argument is a binary expression with Add operator (string concatenation)
	firstArg := mi.Arguments.Elements[0].Element
	bin, ok := firstArg.(*tree.Binary)
	if !ok || bin.Operator.Element != tree.Add {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "possible SQL injection via string concatenation"))
	return mi
}
