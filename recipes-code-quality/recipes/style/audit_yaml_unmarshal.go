/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditYamlUnmarshal finds calls to `yaml.Unmarshal()`. YAML parsing should
// validate input carefully to avoid unexpected behavior from untrusted data.
type AuditYamlUnmarshal struct {
	recipe.Base
}

func (r *AuditYamlUnmarshal) Name() string {
	return "org.openrewrite.golang.codequality.AuditYamlUnmarshal"
}
func (r *AuditYamlUnmarshal) DisplayName() string { return "Audit yaml.Unmarshal" }
func (r *AuditYamlUnmarshal) Description() string {
	return "Find calls to `yaml.Unmarshal()`. YAML parsing should validate input carefully to avoid unexpected behavior from untrusted data."
}
func (r *AuditYamlUnmarshal) Tags() []string { return []string{"style"} }

func (r *AuditYamlUnmarshal) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditYamlUnmarshalVisitor{})
}

type auditYamlUnmarshalVisitor struct {
	visitor.GoVisitor
}

func (v *auditYamlUnmarshalVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "yaml" {
		return mi
	}

	if mi.Name.Name != "Unmarshal" {
		return mi
	}

	mi = mi.WithMarkers(java.MarkupInfo(mi.Markers, "yaml.Unmarshal() call; validate input carefully"))
	return mi
}
