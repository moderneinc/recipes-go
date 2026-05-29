/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// permission777Methods lists the os package methods that accept a file mode.
var permission777Methods = map[string]bool{
	"Chmod":     true,
	"MkdirAll":  true,
	"Mkdir":     true,
	"WriteFile": true,
}

// UseRestrictiveFilePermissions replaces `0777` with `0755` in calls to
// `os.Chmod`, `os.MkdirAll`, `os.Mkdir`, or `os.WriteFile`. Using 0777
// grants full read/write/execute permission to all users, which is overly
// permissive.
type UseRestrictiveFilePermissions struct {
	recipe.Base
}

func (r *UseRestrictiveFilePermissions) Name() string {
	return "org.openrewrite.golang.codequality.UseRestrictiveFilePermissions"
}
func (r *UseRestrictiveFilePermissions) DisplayName() string {
	return "Use restrictive file permissions"
}
func (r *UseRestrictiveFilePermissions) Description() string {
	return "Replace `0777` with `0755` in `os.Chmod`, `os.MkdirAll`, `os.Mkdir`, or `os.WriteFile`. Overly permissive file permissions are a security risk."
}
func (r *UseRestrictiveFilePermissions) Tags() []string { return []string{"style", "security"} }

func (r *UseRestrictiveFilePermissions) Editor() recipe.TreeVisitor {
	return visitor.Init(&useRestrictiveFilePermissionsVisitor{})
}

type useRestrictiveFilePermissionsVisitor struct {
	visitor.GoVisitor
}

func (v *useRestrictiveFilePermissionsVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "os" {
		return mi
	}

	if !permission777Methods[mi.Name.Name] {
		return mi
	}

	newArgs := make([]java.RightPadded[java.Expression], len(mi.Arguments.Elements))
	changed := false
	for i, arg := range mi.Arguments.Elements {
		if lit, ok := arg.Element.(*java.Literal); ok {
			if lit.Source == "0777" {
				newArgs[i] = java.RightPadded[java.Expression]{Element: lit.WithSource("0755"), After: arg.After, Markers: arg.Markers}
				changed = true
				continue
			}
			if lit.Source == "0o777" {
				newArgs[i] = java.RightPadded[java.Expression]{Element: lit.WithSource("0o755"), After: arg.After, Markers: arg.Markers}
				changed = true
				continue
			}
		}
		newArgs[i] = arg
	}

	if !changed {
		return mi
	}

	return &java.MethodInvocation{
		ID:      mi.ID,
		Prefix:  mi.Prefix,
		Markers: mi.Markers,
		Select:  mi.Select,
		Name:    mi.Name,
		Arguments: java.Container[java.Expression]{
			Before:   mi.Arguments.Before,
			Elements: newArgs,
			Markers:  mi.Arguments.Markers,
		},
		MethodType: mi.MethodType,
	}
}
