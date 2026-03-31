/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// osFileOpenMethods lists os package functions that open files.
var osFileOpenMethods = map[string]bool{
	"Open":     true,
	"Create":   true,
	"OpenFile": true,
}

// FindOsFileOpen finds calls to `os.Open()`, `os.Create()`, and
// `os.OpenFile()`. Opened files must be closed — typically via a deferred
// call to Close — to avoid resource leaks.
type FindOsFileOpen struct {
	recipe.Base
}

func (r *FindOsFileOpen) Name() string {
	return "org.openrewrite.golang.codequality.FindOsFileOpen"
}
func (r *FindOsFileOpen) DisplayName() string { return "Find os file open calls" }
func (r *FindOsFileOpen) Description() string {
	return "Find calls to `os.Open`, `os.Create`, and `os.OpenFile`. Ensure the returned file is closed to avoid resource leaks."
}
func (r *FindOsFileOpen) Tags() []string { return []string{"style", "os"} }

func (r *FindOsFileOpen) Editor() recipe.TreeVisitor {
	return visitor.Init(&findOsFileOpenVisitor{})
}

type findOsFileOpenVisitor struct {
	visitor.GoVisitor
}

func (v *findOsFileOpenVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "os" {
		return mi
	}

	if !osFileOpenMethods[mi.Name.Name] {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "ensure the file is closed"))
	return mi
}
