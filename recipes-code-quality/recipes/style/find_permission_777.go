/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// permission777Methods lists the os package methods that accept a file mode.
var permission777Methods = map[string]bool{
	"Chmod":    true,
	"MkdirAll": true,
	"Mkdir":    true,
	"WriteFile": true,
}

// FindPermission777 finds calls to `os.Chmod`, `os.MkdirAll`, `os.Mkdir`, or
// `os.WriteFile` with permission `0777` or `0o777`. Using 0777 grants full
// read/write/execute permission to all users, which is overly permissive.
type FindPermission777 struct {
	recipe.Base
}

func (r *FindPermission777) Name() string {
	return "org.openrewrite.golang.codequality.FindPermission777"
}
func (r *FindPermission777) DisplayName() string { return "Find permission 0777" }
func (r *FindPermission777) Description() string {
	return "Find `os.Chmod`, `os.MkdirAll`, `os.Mkdir`, or `os.WriteFile` with permission 0777. Overly permissive file permissions are a security risk."
}
func (r *FindPermission777) Tags() []string { return []string{"style", "security"} }

func (r *FindPermission777) Editor() recipe.TreeVisitor {
	return visitor.Init(&findPermission777Visitor{})
}

type findPermission777Visitor struct {
	visitor.GoVisitor
}

func (v *findPermission777Visitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "os" {
		return mi
	}

	if !permission777Methods[mi.Name.Name] {
		return mi
	}

	for _, arg := range mi.Arguments.Elements {
		lit, ok := arg.Element.(*tree.Literal)
		if !ok {
			continue
		}
		if lit.Source == "0777" || lit.Source == "0o777" {
			mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "overly permissive file permission 0777; consider using a more restrictive mode"))
			return mi
		}
	}

	return mi
}
