/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditContextBackground finds calls to `context.Background()`. In most
// application code the context should be propagated from the caller rather
// than creating a new empty context via `context.Background()`.
type AuditContextBackground struct {
	recipe.Base
}

func (r *AuditContextBackground) Name() string {
	return "org.openrewrite.golang.codequality.AuditContextBackground"
}
func (r *AuditContextBackground) DisplayName() string { return "Audit context.Background" }
func (r *AuditContextBackground) Description() string {
	return "Find calls to `context.Background()`. Consider using a context passed from the caller instead."
}
func (r *AuditContextBackground) Tags() []string { return []string{"style"} }

func (r *AuditContextBackground) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditContextBackgroundVisitor{})
}

// The matcher resolves the receiver through the type system, so a local named
// `context` with a Background method of its own is not a match.
var contextBackgroundMatcher = matcher.NewMethodMatcher("context Background()")

type auditContextBackgroundVisitor struct {
	visitor.GoVisitor
}

func (v *auditContextBackgroundVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if !contextBackgroundMatcher.Matches(mi) {
		return mi
	}

	mi = mi.WithMarkers(java.MarkupInfo(mi.Markers, "context.Background() call; consider using a passed context instead"))
	return mi
}
