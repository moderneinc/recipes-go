/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindTempFile finds calls to `os.CreateTemp` and `os.MkdirTemp`. Temporary
// files and directories should be cleaned up when no longer needed.
type FindTempFile struct {
	recipe.Base
}

func (r *FindTempFile) Name() string {
	return "org.openrewrite.golang.codequality.FindTempFile"
}
func (r *FindTempFile) DisplayName() string { return "Find temp file creation" }
func (r *FindTempFile) Description() string {
	return "Find calls to `os.CreateTemp` and `os.MkdirTemp`. Temporary files and directories should be cleaned up when no longer needed."
}
func (r *FindTempFile) Tags() []string { return []string{"style", "resource-management"} }

func (r *FindTempFile) Editor() recipe.TreeVisitor {
	return visitor.Init(&findTempFileVisitor{})
}

type findTempFileVisitor struct {
	visitor.GoVisitor
}

var tempFileMethods = map[string]bool{
	"CreateTemp": true,
	"MkdirTemp":  true,
}

func (v *findTempFileVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "os" {
		return mi
	}

	if !tempFileMethods[mi.Name.Name] {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "ensure temp file or directory is cleaned up"))
	return mi
}
