/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseErrorMethod replaces `fmt.Sprint(err)` calls with `err.Error()`.
// Converting an error to a string via fmt.Sprint is unclear — use
// `err.Error()` for direct access or `fmt.Errorf` for wrapping with context.
type UseErrorMethod struct {
	recipe.Base
}

func (r *UseErrorMethod) Name() string {
	return "org.openrewrite.golang.codequality.UseErrorMethod"
}
func (r *UseErrorMethod) DisplayName() string { return "Use .Error() method" }
func (r *UseErrorMethod) Description() string {
	return "Replace `fmt.Sprint(err)` with `err.Error()` for clarity."
}
func (r *UseErrorMethod) Tags() []string { return []string{"error-handling", "lint"} }

func (r *UseErrorMethod) Editor() recipe.TreeVisitor {
	return visitor.Init(&useErrorMethodVisitor{})
}

type useErrorMethodVisitor struct {
	visitor.GoVisitor
}

func (v *useErrorMethodVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "fmt" {
		return mi
	}

	if mi.Name.Name != "Sprint" {
		return mi
	}

	// Check for exactly 1 real argument that is an identifier named "err".
	args := realArgs(mi.Arguments.Elements)
	if len(args) != 1 {
		return mi
	}

	argIdent, ok := args[0].Element.(*tree.Identifier)
	if !ok || argIdent.Name != "err" {
		return mi
	}

	// Build err.Error() as a replacement, preserving the original leading prefix.
	errIdent := argIdent.WithPrefix(ident.Prefix)
	errorName := &tree.Identifier{Name: "Error"}
	return &tree.MethodInvocation{
		Prefix:    mi.Prefix,
		Select:    &tree.RightPadded[tree.Expression]{Element: errIdent},
		Name:      errorName,
		Arguments: tree.Container[tree.Expression]{},
	}
}

// realArgs returns arguments that are not *tree.Empty.
func realArgs(args []tree.RightPadded[tree.Expression]) []tree.RightPadded[tree.Expression] {
	var out []tree.RightPadded[tree.Expression]
	for _, a := range args {
		if _, isEmpty := a.Element.(*tree.Empty); !isEmpty {
			out = append(out, a)
		}
	}
	return out
}
