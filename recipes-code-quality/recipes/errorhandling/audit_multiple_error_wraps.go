/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AuditMultipleErrorWraps replaces extra `%w` verbs in `fmt.Errorf` format
// strings with `%v`, keeping only the first `%w`. Multiple `%w` is invalid in
// Go < 1.20 and, while technically supported in 1.20+, wrapping multiple errors
// is unusual and often unintentional.
type AuditMultipleErrorWraps struct {
	recipe.Base
}

func (r *AuditMultipleErrorWraps) Name() string {
	return "org.openrewrite.golang.codequality.AuditMultipleErrorWraps"
}
func (r *AuditMultipleErrorWraps) DisplayName() string {
	return "Replace extra %w verbs with %v in fmt.Errorf"
}
func (r *AuditMultipleErrorWraps) Description() string {
	return "Replace all but the first `%w` with `%v` in `fmt.Errorf` format strings. Multiple error wrapping is invalid in Go < 1.20 and rare in later versions."
}
func (r *AuditMultipleErrorWraps) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *AuditMultipleErrorWraps) Editor() recipe.TreeVisitor {
	return visitor.Init(&auditMultipleErrorWrapsVisitor{})
}

type auditMultipleErrorWrapsVisitor struct {
	visitor.GoVisitor
}

func (v *auditMultipleErrorWrapsVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	// Match: fmt.Errorf(...)
	if mi.Select == nil {
		return mi
	}
	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "fmt" {
		return mi
	}
	if mi.Name.Name != "Errorf" {
		return mi
	}

	// Need at least one argument: the format string.
	args := mi.Arguments.Elements
	if len(args) < 1 {
		return mi
	}

	// First argument must be a string literal.
	fmtLit, ok := args[0].Element.(*tree.Literal)
	if !ok || fmtLit.Kind != tree.StringLiteral {
		return mi
	}

	// Only act when there are multiple %w verbs.
	if strings.Count(fmtLit.Source, "%w") <= 1 {
		return mi
	}

	// Replace all %w after the first with %v.
	newSource := replaceExtraW(fmtLit.Source)

	newFmtLit := fmtLit.WithSource(newSource)

	// Rebuild the arguments with the modified format literal.
	newArgs := make([]tree.RightPadded[tree.Expression], len(args))
	copy(newArgs, args)
	newArgs[0] = tree.RightPadded[tree.Expression]{
		Element: newFmtLit,
		After:   args[0].After,
		Markers: args[0].Markers,
	}

	newArgContainer := mi.Arguments
	newArgContainer.Elements = newArgs

	return &tree.MethodInvocation{
		ID:        mi.ID,
		Prefix:    mi.Prefix,
		Markers:   mi.Markers,
		Select:    mi.Select,
		Name:      mi.Name,
		Arguments: newArgContainer,
	}
}

// replaceExtraW replaces all occurrences of "%w" after the first with "%v".
func replaceExtraW(s string) string {
	first := true
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if i+1 < len(s) && s[i] == '%' && s[i+1] == 'w' {
			if first {
				b.WriteString("%w")
				first = false
			} else {
				b.WriteString("%v")
			}
			i++ // skip the 'w'
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
