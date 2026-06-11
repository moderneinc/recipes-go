/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditTestFatal finds `t.Fatal()` and `t.Fatalf()` calls. These methods
// abort the test immediately and, when called from a goroutine other than
// the test function's goroutine (Go 1.16+), cause a panic. Consider using
// `t.Error`/`t.Errorf` instead, especially inside goroutines.
type AuditTestFatal struct {
	recipe.Base
}

func (r *AuditTestFatal) Name() string {
	return "org.openrewrite.golang.codequality.AuditTestFatal"
}
func (r *AuditTestFatal) DisplayName() string { return "Audit test fatal" }
func (r *AuditTestFatal) Description() string {
	return "Find `t.Fatal()` and `t.Fatalf()` calls. These abort the test immediately and panic when called from a goroutine other than the test function's goroutine."
}
func (r *AuditTestFatal) Tags() []string { return []string{"testing"} }

func (r *AuditTestFatal) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditTestFatalVisitor{})
}

type auditTestFatalVisitor struct {
	visitor.GoVisitor
}

func (v *auditTestFatalVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "t" {
		return mi
	}

	if !strings.HasPrefix(mi.Name.Name, "Fatal") {
		return mi
	}

	mi = mi.WithMarkers(
		java.MarkupInfo(mi.Markers, "t.Fatal call found; consider t.Error in goroutines"),
	)
	return mi
}
