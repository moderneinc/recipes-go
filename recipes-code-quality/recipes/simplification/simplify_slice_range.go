/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
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

func (v *simplifySliceRangeVisitor) VisitSlice(s *golang.Slice, p any) java.J {
	s = v.GoVisitor.VisitSlice(s, p).(*golang.Slice)

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
	empty := &java.Empty{}
	return &golang.Slice{
		ID:           s.ID,
		Prefix:       s.Prefix,
		Markers:      s.Markers,
		Indexed:      s.Indexed,
		OpenBracket:  s.OpenBracket,
		Low:          java.RightPadded[java.Expression]{Element: empty, After: s.Low.After},
		High:         java.RightPadded[java.Expression]{Element: empty, After: s.High.After},
		Max:          nil,
		CloseBracket: s.CloseBracket,
	}
}

func identNameFromExpr(expr java.Expression) string {
	switch n := expr.(type) {
	case *java.Identifier:
		return n.Name
	default:
		return ""
	}
}

func isLiteralZero(expr java.Expression) bool {
	lit, ok := expr.(*java.Literal)
	return ok && lit.Source == "0"
}

func isLenOfVar(expr java.Expression, varName string) bool {
	mi, ok := expr.(*java.MethodInvocation)
	if !ok || mi.Select != nil || mi.Name.Name != "len" {
		return false
	}
	for _, arg := range mi.Arguments.Elements {
		if ident, ok := arg.Element.(*java.Identifier); ok {
			return ident.Name == varName
		}
	}
	return false
}
