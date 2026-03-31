/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"strings"

	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindLogFatal finds calls to `log.Fatal()` and `log.Fatalf()`. These call
// os.Exit(1) which does not run deferred functions. Often a code smell in
// library code.
type FindLogFatal struct {
	recipe.Base
}

func (r *FindLogFatal) Name() string {
	return "org.openrewrite.golang.codequality.FindLogFatal"
}
func (r *FindLogFatal) DisplayName() string { return "Find log.Fatal calls" }
func (r *FindLogFatal) Description() string {
	return "Find `log.Fatal` and `log.Fatalf` calls which call os.Exit(1) and do not run deferred functions."
}
func (r *FindLogFatal) Tags() []string { return []string{"error-handling", "lint"} }

func (r *FindLogFatal) Editor() recipe.TreeVisitor {
	return visitor.Init(&findLogFatalVisitor{})
}

type findLogFatalVisitor struct {
	visitor.GoVisitor
}

func (v *findLogFatalVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "log" {
		return mi
	}

	if !strings.HasPrefix(mi.Name.Name, "Fatal") {
		return mi
	}

	mi = mi.WithMarkers(
		tree.FoundSearchResult(mi.Markers, "log.Fatal calls os.Exit(1) and does not run deferred functions"),
	)
	return mi
}
