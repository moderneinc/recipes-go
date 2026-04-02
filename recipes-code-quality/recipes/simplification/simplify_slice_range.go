/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/moderneinc/recipes-go/code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// SimplifySliceRange replaces `s[0:len(s)]` with `s[:]`.
// Staticcheck: S1003
type SimplifySliceRange struct {
	recipe.Base
}

func (r *SimplifySliceRange) Name() string {
	return "org.openrewrite.golang.codequality.SimplifySliceRange"
}
func (r *SimplifySliceRange) DisplayName() string { return "Simplify slice range" }
func (r *SimplifySliceRange) Description() string {
	return "Replace `s[0:len(s)]` with `s[:]`."
}
func (r *SimplifySliceRange) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *SimplifySliceRange) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1003", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *SimplifySliceRange) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifySliceRangeVisitor{})
}

type simplifySliceRangeVisitor struct {
	visitor.GoVisitor
}

func (v *simplifySliceRangeVisitor) VisitSlice(s *tree.Slice, p any) tree.J {
	s = v.GoVisitor.VisitSlice(s, p).(*tree.Slice)

	// Must be a 2-index slice (no Max)
	if s.Max != nil {
		return s
	}

	sliceName := identNameFromExpr(s.Indexed)
	if sliceName == "" {
		return s
	}

	// Low must be literal 0
	if !isLiteralZero(s.Low.Element) {
		return s
	}

	// High must be len(sliceName)
	if !isLenOfVar(s.High.Element, sliceName) {
		return s
	}

	// Replace low with Empty and high with Empty to produce s[:]
	empty := &tree.Empty{}
	return &tree.Slice{
		ID:           s.ID,
		Prefix:       s.Prefix,
		Markers:      s.Markers,
		Indexed:      s.Indexed,
		OpenBracket:  s.OpenBracket,
		Low:          tree.RightPadded[tree.Expression]{Element: empty, After: s.Low.After},
		High:         tree.RightPadded[tree.Expression]{Element: empty, After: s.High.After},
		Max:          nil,
		CloseBracket: s.CloseBracket,
	}
}

func identNameFromExpr(expr tree.Expression) string {
	switch n := expr.(type) {
	case *tree.Identifier:
		return n.Name
	default:
		return ""
	}
}

func isLiteralZero(expr tree.Expression) bool {
	lit, ok := expr.(*tree.Literal)
	return ok && lit.Source == "0"
}

func isLenOfVar(expr tree.Expression, varName string) bool {
	mi, ok := expr.(*tree.MethodInvocation)
	if !ok || mi.Select != nil || mi.Name.Name != "len" {
		return false
	}
	for _, arg := range mi.Arguments.Elements {
		if ident, ok := arg.Element.(*tree.Identifier); ok {
			return ident.Name == varName
		}
	}
	return false
}
