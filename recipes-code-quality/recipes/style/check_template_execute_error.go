/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// CheckTemplateExecuteError finds calls to `template.Execute` and
// `template.ExecuteTemplate`. These calls return an error that should be
// checked to avoid silent rendering failures.
type CheckTemplateExecuteError struct {
	recipe.Base
}

func (r *CheckTemplateExecuteError) Name() string {
	return "org.openrewrite.golang.codequality.CheckTemplateExecuteError"
}
func (r *CheckTemplateExecuteError) DisplayName() string { return "Check template execute error" }
func (r *CheckTemplateExecuteError) Description() string {
	return "Find calls to `Execute` and `ExecuteTemplate` on templates. The returned error should be checked to avoid silent rendering failures."
}
func (r *CheckTemplateExecuteError) Tags() []string { return []string{"style", "html/template"} }

func (r *CheckTemplateExecuteError) Editor() recipe.TreeVisitor {
	return visitor.Init(&checkTemplateExecuteErrorVisitor{})
}

type checkTemplateExecuteErrorVisitor struct {
	visitor.GoVisitor
}

func (v *checkTemplateExecuteErrorVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	if mi.Name.Name != "Execute" && mi.Name.Name != "ExecuteTemplate" {
		return mi
	}

	mi = mi.WithMarkers(tree.MarkupInfo(mi.Markers, "ensure template execute error is checked"))
	return mi
}
