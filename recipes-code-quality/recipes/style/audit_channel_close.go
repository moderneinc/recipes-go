/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditChannelClose finds calls to the built-in `close()` function. Closing a
// channel should only be done by the sender, and double-closing a channel
// causes a panic. This recipe highlights all close calls for review.
type AuditChannelClose struct {
	recipe.Base
}

func (r *AuditChannelClose) Name() string {
	return "org.openrewrite.golang.codequality.AuditChannelClose"
}
func (r *AuditChannelClose) DisplayName() string { return "Audit channel close" }
func (r *AuditChannelClose) Description() string {
	return "Find calls to the built-in `close()` function. Channels should only be closed by the sender, and double-closing causes a panic."
}
func (r *AuditChannelClose) Tags() []string { return []string{"style", "concurrency"} }

func (r *AuditChannelClose) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditChannelCloseVisitor{})
}

type auditChannelCloseVisitor struct {
	visitor.GoVisitor
}

func (v *auditChannelCloseVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	// Match: close(...) — built-in, so no Select and Name == "close".
	if mi.Select != nil || mi.Name.Name != "close" {
		return mi
	}

	mi = mi.WithMarkers(java.MarkupInfo(mi.Markers, "ensure channel is only closed by the sender"))
	return mi
}
