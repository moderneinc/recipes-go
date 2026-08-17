/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/moderneinc/recipes-go/diagnostic"
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
	return "Remove a conversion whose operand already has the target type, such as `string(s)` where `s` is a `string`. Conversions to `int`, `int8`, `byte`, `rune`, `float64` and the complex types are left alone."
}
func (r *RemoveRedundantTypeConversion) Tags() []string { return []string{"cleanup"} }

func (r *RemoveRedundantTypeConversion) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1021", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *RemoveRedundantTypeConversion) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeRedundantTypeConversionVisitor{})
}

type removeRedundantTypeConversionVisitor struct {
	visitor.GoVisitor
}

func (v *removeRedundantTypeConversionVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	// A conversion reads as a call with no receiver and one argument.
	if mi.Select != nil {
		return mi
	}
	attributedType, convertible := unambiguousBuiltins[mi.Name.Name]
	if !convertible {
		return mi
	}

	var operand java.Expression
	for _, a := range mi.Arguments.Elements {
		if _, isEmpty := a.Element.(*java.Empty); isEmpty {
			continue
		}
		if operand != nil {
			return mi
		}
		operand = a.Element
	}
	if operand == nil {
		return mi
	}

	// An untyped constant takes its type from the conversion, so the conversion
	// is what gives it one.
	if _, isLiteral := operand.(*java.Literal); isLiteral {
		return mi
	}
	if matcher.GetFullyQualifiedName(matcher.TypeOfExpression(operand)) != attributedType {
		return mi
	}
	// The operand inherits the conversion's prefix, which prependExprPrefix
	// carries over for these forms.
	switch operand.(type) {
	case *java.Identifier, *java.FieldAccess, *java.MethodInvocation:
		return prependExprPrefix(operand, mi.Prefix)
	}
	return mi
}

// unambiguousBuiltins maps a Go builtin type to the type attributed to an
// expression of that type. Only the builtins that own their attributed type
// outright are listed: `int` shares one with `int32`, `byte` with `int8`, and
// `float64` with the complex types, so a match there would not establish that
// the operand and the conversion agree.
var unambiguousBuiltins = map[string]string{
	"string":  "String",
	"bool":    "boolean",
	"int16":   "short",
	"int64":   "long",
	"float32": "float",
	"uint":    "uint",
	"uint8":   "uint8",
	"uint16":  "uint16",
	"uint32":  "uint32",
	"uint64":  "uint64",
	"uintptr": "uintptr",
}
