/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindYamlUnmarshal finds calls to `yaml.Unmarshal()`. YAML parsing should
// validate input carefully to avoid unexpected behavior from untrusted data.
type FindYamlUnmarshal struct {
	recipe.Base
}

func (r *FindYamlUnmarshal) Name() string {
	return "org.openrewrite.golang.codequality.FindYamlUnmarshal"
}
func (r *FindYamlUnmarshal) DisplayName() string { return "Find yaml.Unmarshal() calls" }
func (r *FindYamlUnmarshal) Description() string {
	return "Find calls to `yaml.Unmarshal()`. YAML parsing should validate input carefully to avoid unexpected behavior from untrusted data."
}
func (r *FindYamlUnmarshal) Tags() []string { return []string{"style"} }

func (r *FindYamlUnmarshal) Editor() recipe.TreeVisitor {
	return visitor.Init(&findYamlUnmarshalVisitor{})
}

type findYamlUnmarshalVisitor struct {
	visitor.GoVisitor
}

func (v *findYamlUnmarshalVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "yaml" {
		return mi
	}

	if mi.Name.Name != "Unmarshal" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "yaml.Unmarshal() call; validate input carefully"))
	return mi
}
