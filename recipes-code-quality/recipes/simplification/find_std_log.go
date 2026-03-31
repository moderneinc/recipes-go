/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"strings"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindStdLog finds calls to the standard `log` package such as `log.Println`,
// `log.Printf`, `log.Fatal`, and `log.Fatalf`. In Go 1.21+ consider migrating
// to `log/slog` for structured logging.
type FindStdLog struct {
	recipe.Base
}

func (r *FindStdLog) Name() string {
	return "org.openrewrite.golang.codequality.FindStdLog"
}
func (r *FindStdLog) DisplayName() string { return "Find standard log usage" }
func (r *FindStdLog) Description() string {
	return "Find calls to the standard `log` package (`log.Print*`, `log.Fatal*`). Consider migrating to `log/slog` for structured logging (Go 1.21+)."
}
func (r *FindStdLog) Tags() []string { return []string{"simplification", "logging"} }

func (r *FindStdLog) Editor() recipe.TreeVisitor {
	return visitor.Init(&findStdLogVisitor{})
}

type findStdLogVisitor struct {
	visitor.GoVisitor
}

// stdLogPrefixes lists the method-name prefixes on the standard `log` package
// that should be flagged.
var stdLogPrefixes = []string{"Print", "Fatal"}

func (v *findStdLogVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "log" {
		return mi
	}

	for _, prefix := range stdLogPrefixes {
		if strings.HasPrefix(mi.Name.Name, prefix) {
			mi = mi.WithMarkers(
				tree.FoundSearchResult(mi.Markers, "consider migrating to log/slog for structured logging (Go 1.21+)"),
			)
			return mi
		}
	}

	return mi
}
