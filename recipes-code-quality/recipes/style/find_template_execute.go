/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindTemplateExecute finds calls to `template.Execute` and
// `template.ExecuteTemplate`. These calls return an error that should be
// checked to avoid silent rendering failures.
type FindTemplateExecute struct {
	recipe.Base
}

func (r *FindTemplateExecute) Name() string {
	return "org.openrewrite.golang.codequality.FindTemplateExecute"
}
func (r *FindTemplateExecute) DisplayName() string { return "Find template Execute calls" }
func (r *FindTemplateExecute) Description() string {
	return "Find calls to `Execute` and `ExecuteTemplate` on templates. The returned error should be checked to avoid silent rendering failures."
}
func (r *FindTemplateExecute) Tags() []string { return []string{"style", "html/template"} }

func (r *FindTemplateExecute) Editor() recipe.TreeVisitor {
	return visitor.Init(&findTemplateExecuteVisitor{})
}

type findTemplateExecuteVisitor struct {
	visitor.GoVisitor
}

func (v *findTemplateExecuteVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	if mi.Name.Name != "Execute" && mi.Name.Name != "ExecuteTemplate" {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "ensure template execute error is checked"))
	return mi
}
