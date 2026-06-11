/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// sqlQueryMethods lists database methods that accept SQL strings.
var sqlQueryMethods = map[string]bool{
	"Query":    true,
	"QueryRow": true,
	"Exec":     true,
}

// UseParameterizedSqlQuery finds SQL queries built with string concatenation, e.g.
// `db.Query("SELECT * FROM " + table)`. Building SQL via concatenation is a
// common source of SQL injection vulnerabilities. Use parameterized queries
// instead.
type UseParameterizedSqlQuery struct {
	recipe.Base
}

func (r *UseParameterizedSqlQuery) Name() string {
	return "org.openrewrite.golang.codequality.UseParameterizedSqlQuery"
}
func (r *UseParameterizedSqlQuery) DisplayName() string { return "Use parameterized SQL queries" }
func (r *UseParameterizedSqlQuery) Description() string {
	return "Find SQL queries built with string concatenation via db.Query, db.QueryRow, or db.Exec. Use parameterized queries to avoid SQL injection."
}
func (r *UseParameterizedSqlQuery) Tags() []string { return []string{"security", "database/sql"} }

func (r *UseParameterizedSqlQuery) Editor() recipe.TreeVisitor {
	return visitor.Init(&useParameterizedSqlQueryVisitor{})
}

type useParameterizedSqlQueryVisitor struct {
	visitor.GoVisitor
}

func (v *useParameterizedSqlQueryVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

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
	bin, ok := firstArg.(*java.Binary)
	if !ok || bin.Operator.Element != java.Add {
		return mi
	}

	mi = mi.WithMarkers(java.MarkupWarn(mi.Markers, "possible SQL injection via string concatenation"))
	return mi
}
