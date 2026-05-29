/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditTestMain finds `func TestMain(m *testing.M)` declarations. TestMain
// overrides the default test execution, which can affect all tests in the
// package. Flag these for awareness during code review.
type AuditTestMain struct {
	recipe.Base
}

func (r *AuditTestMain) Name() string {
	return "org.openrewrite.golang.codequality.AuditTestMain"
}
func (r *AuditTestMain) DisplayName() string { return "Audit TestMain" }
func (r *AuditTestMain) Description() string {
	return "Find `func TestMain(m *testing.M)` declarations. TestMain overrides the default test execution for the entire package."
}
func (r *AuditTestMain) Tags() []string { return []string{"testing"} }

func (r *AuditTestMain) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditTestMainVisitor{})
}

type auditTestMainVisitor struct {
	visitor.GoVisitor
}

func (v *auditTestMainVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	md = v.GoVisitor.VisitMethodDeclaration(md, p).(*java.MethodDeclaration)

	if md.Name == nil || md.Name.Name != "TestMain" {
		return md
	}

	// Must be a free function (no receiver).
	if md.Receiver != nil {
		return md
	}

	md = md.WithName(md.Name.WithMarkers(
		java.MarkupInfo(md.Name.Markers, "TestMain overrides default test execution"),
	))
	return md
}
