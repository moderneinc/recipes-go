/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveRedundantTypeConversion removes a conversion whose operand already has
// the target type, such as `string(s)` for a `string` s.
type RemoveRedundantTypeConversion struct {
	recipe.Base
}

func (r *RemoveRedundantTypeConversion) Name() string {
	return "org.openrewrite.golang.codequality.RemoveRedundantTypeConversion"
}
func (r *RemoveRedundantTypeConversion) DisplayName() string {
	return "Remove redundant type conversion"
}
func (r *RemoveRedundantTypeConversion) Description() string {
	return "Remove a conversion whose operand already has the target type, such as `string(s)` where `s` is a `string`. `byte` and `uint8` name one type, as do `rune` and `int32`, so a conversion between either pair is removed too."
}
func (r *RemoveRedundantTypeConversion) Tags() []string { return []string{"cleanup"} }

func (r *RemoveRedundantTypeConversion) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "unconvert", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *RemoveRedundantTypeConversion) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeRedundantTypeConversionVisitor{})
}

type removeRedundantTypeConversionVisitor struct {
	visitor.GoVisitor
}

func (v *removeRedundantTypeConversionVisitor) VisitTypeCast(tc *java.TypeCast, p any) java.J {
	tc = v.GoVisitor.VisitTypeCast(tc, p).(*java.TypeCast)

	if tc.Clazz == nil {
		return tc
	}
	target, isBuiltin := tc.Clazz.Tree.Element.(*java.Identifier)
	if !isBuiltin {
		return tc
	}
	if !lstutil.GoBasicTypes[target.Name] {
		return tc
	}

	operand := tc.Expr
	if operand == nil {
		return tc
	}
	// IsSameGoType resolves `byte`/`uint8` and `rune`/`int32` to one type, and
	// answers false for a literal, whose untyped constant takes its type from
	// the conversion being removed.
	if !matcher.IsSameGoType(target.Type, matcher.TypeOfExpression(operand)) {
		return tc
	}
	// The operand inherits the conversion's prefix, which prependExprPrefix
	// carries over for these forms.
	switch operand.(type) {
	case *java.Identifier, *java.FieldAccess, *java.MethodInvocation:
		return prependExprPrefix(operand, tc.Prefix)
	}
	return tc
}
