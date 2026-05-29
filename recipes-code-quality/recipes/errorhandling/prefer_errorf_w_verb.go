/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// PreferErrorfWrapVerb replaces `fmt.Errorf("...: %s", err)` with
// `fmt.Errorf("...: %w", err)`. Using `%s` loses the error chain, which breaks
// `errors.Is` and `errors.As` unwrapping.
type PreferErrorfWrapVerb struct {
	recipe.Base
}

func (r *PreferErrorfWrapVerb) Name() string {
	return "org.openrewrite.golang.codequality.PreferErrorfWrapVerb"
}
func (r *PreferErrorfWrapVerb) DisplayName() string {
	return "Prefer %w over %s in fmt.Errorf for error wrapping"
}
func (r *PreferErrorfWrapVerb) Description() string {
	return "Replace `%s` with `%w` in `fmt.Errorf` format strings when the corresponding argument is an error, to preserve the error chain."
}
func (r *PreferErrorfWrapVerb) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *PreferErrorfWrapVerb) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferErrorfWrapVerbVisitor{})
}

type preferErrorfWrapVerbVisitor struct {
	visitor.GoVisitor
}

func (v *preferErrorfWrapVerbVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	// Match: fmt.Errorf(...)
	if mi.Select == nil {
		return mi
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "fmt" {
		return mi
	}
	if mi.Name.Name != "Errorf" {
		return mi
	}

	// Need at least 2 arguments: format string and at least one arg.
	args := mi.Arguments.Elements
	if len(args) < 2 {
		return mi
	}

	// First argument must be a string literal containing %s (but not %w).
	fmtLit, ok := args[0].Element.(*java.Literal)
	if !ok || fmtLit.Kind != java.StringLiteral {
		return mi
	}
	content := fmtLit.Source
	if !strings.Contains(content, "%s") || strings.Contains(content, "%w") {
		return mi
	}

	// Last real argument should be an identifier named "err".
	lastArg := args[len(args)-1].Element
	lastIdent, ok := lastArg.(*java.Identifier)
	if !ok || lastIdent.Name != "err" {
		return mi
	}

	// Replace the last occurrence of %s with %w in the format string source.
	lastIdx := strings.LastIndex(content, "%s")
	newSource := content[:lastIdx] + "%w" + content[lastIdx+2:]
	newFmtLit := fmtLit.WithSource(newSource)

	// Rebuild the arguments with the modified format literal.
	newArgs := make([]java.RightPadded[java.Expression], len(args))
	copy(newArgs, args)
	newArgs[0] = java.RightPadded[java.Expression]{
		Element: newFmtLit,
		After:   args[0].After,
		Markers: args[0].Markers,
	}

	newArgContainer := mi.Arguments
	newArgContainer.Elements = newArgs

	return &java.MethodInvocation{
		ID:        mi.ID,
		Prefix:    mi.Prefix,
		Markers:   mi.Markers,
		Select:    mi.Select,
		Name:      mi.Name,
		Arguments: newArgContainer,
	}
}
