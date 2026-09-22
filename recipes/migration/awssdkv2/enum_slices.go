/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"
	"sync"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// A field holding a list of enum values changed twice between v1 and v2: the
// list went from pointers to values, and the element from *string to the named
// enum type. The value-slice helpers do not reach it — they preserve the v1
// element type, which is the half that also has to change — so the literal is
// rewritten in place instead.
const enumSliceSource = `package p

var _ = []%s.%s{}
`

var (
	enumSliceMu    sync.Mutex
	enumSliceCache = map[string]java.Expression{}
)

// enumSliceType returns the `[]<types>.<Enum>` type expression, parsed rather
// than assembled so the slice and the qualified name come out as the parser
// would have made them.
func enumSliceType(typesAlias, enum string) (java.Expression, bool) {
	key := typesAlias + "." + enum

	enumSliceMu.Lock()
	defer enumSliceMu.Unlock()
	if typeExpr, ok := enumSliceCache[key]; ok {
		return typeExpr, typeExpr != nil
	}
	enumSliceCache[key] = nil

	cu, err := parser.NewGoParser().Parse("slice.go", fmt.Sprintf(enumSliceSource, typesAlias, enum))
	if err != nil {
		return nil, false
	}
	lift := visitor.Init(&sliceTypeLift{})
	lift.Visit(cu, nil)
	if lift.typeExpr == nil {
		return nil, false
	}
	enumSliceCache[key] = lift.typeExpr
	return lift.typeExpr, true
}

type sliceTypeLift struct {
	visitor.GoVisitor
	typeExpr java.Expression
}

func (v *sliceTypeLift) VisitComposite(comp *golang.Composite, p any) java.J {
	if v.typeExpr == nil {
		v.typeExpr = comp.TypeExpr
	}
	return v.GoVisitor.VisitComposite(comp, p)
}

// enumSliceOf names the enum a field holds a list of, or "". Both records have
// to agree: the field carries an enum, and v2 holds its list by value.
func enumSliceOf(service, shape, field string) string {
	enum := awsmanifest.FieldEnum(service, shape, field)
	if enum == "" || !awsmanifest.IsValueSlice(service, shape, field) {
		return ""
	}
	return enum
}

// enumSliceLiteral returns the list literal an enum-slice field is set from.
// v1 modelled the list as []*string, so a call site wrote either that literal
// directly or aws.StringSlice around a []string.
func (s *fileScan) enumSliceLiteral(expr java.Expression) (*golang.Composite, bool) {
	if mi, isCall := expr.(*java.MethodInvocation); isCall {
		if helper, isAws := qualifiedCall(mi, s.awsPkg); !isAws || helper != "StringSlice" {
			return nil, false
		}
		args := realArgs(mi)
		if len(args) != 1 {
			return nil, false
		}
		expr = args[0]
	}
	comp, isComposite := expr.(*golang.Composite)
	if !isComposite {
		return nil, false
	}
	return comp, true
}

// retypableEnumSlice reports whether every element of the list can be carried
// into the enum v2 types it with.
func (s *fileScan) retypableEnumSlice(expr java.Expression, service, enum string) bool {
	comp, isLiteral := s.enumSliceLiteral(expr)
	if !isLiteral {
		return false
	}
	for _, rp := range comp.Elements.Elements {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		if !s.retypableEnumElement(rp.Element, service, enum) {
			return false
		}
	}
	return true
}

// retypableEnumElement reports whether one element of such a list carries over.
// Inside aws.StringSlice the element is already a bare string, where inside a
// []*string literal it wears the pointer v1 needed.
func (s *fileScan) retypableEnumElement(expr java.Expression, service, enum string) bool {
	if inner, wrapped := s.enumValueSource(expr); wrapped {
		expr = inner
	}
	fa, isField := expr.(*java.FieldAccess)
	if !isField || fa.Name.Element == nil {
		// A string, or a value computed at runtime, which converts cleanly.
		return true
	}
	target, isIdent := fa.Target.(*java.Identifier)
	if !isIdent || s.services[target.Name] != service {
		return true
	}
	return awsmanifest.EnumOf(service, fa.Name.Element.Name) == enum
}

// enumSliceValue rewrites `[]*string{…}` or `aws.StringSlice([]string{…})` as
// the `[]<types>.<Enum>{…}` v2 takes.
func (v *migrateVisitor) enumSliceValue(expr java.Expression, service, enum string) (java.Expression, bool) {
	comp, isLiteral := v.scan.enumSliceLiteral(expr)
	if !isLiteral {
		return nil, false
	}
	alias := awsmanifest.TypesAlias(service)
	typeExpr, ok := enumSliceType(alias, enum)
	if !ok {
		return nil, false
	}
	v.needsTypes[service] = true
	v.typesAliases[alias] = true

	elements := make([]java.RightPadded[java.Expression], len(comp.Elements.Elements))
	copy(elements, comp.Elements.Elements)
	for i, rp := range elements {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		retyped, rewrote := v.enumElementValue(rp.Element, service, enum)
		if !rewrote {
			return nil, false
		}
		elements[i].Element = retyped
	}

	c := *comp
	c.Prefix = expr.GetPrefix()
	c.TypeExpr = lstutil.SetExprPrefix(typeExpr, java.EmptySpace)
	c.Elements = comp.Elements
	c.Elements.Elements = elements
	return &c, true
}

// enumElementValue retypes one element of such a list. An element already
// wearing a pointer helper goes through the same path a scalar enum field does;
// a bare one is converted where it is not already a constant of the enum.
func (v *migrateVisitor) enumElementValue(expr java.Expression, service, enum string) (java.Expression, bool) {
	if _, wrapped := v.scan.enumValueSource(expr); wrapped {
		return v.enumFieldValue(expr, service, enum)
	}
	if fa, isField := expr.(*java.FieldAccess); isField && fa.Name.Element != nil {
		if target, isIdent := fa.Target.(*java.Identifier); isIdent && v.typesAliases[target.Name] {
			if awsmanifest.EnumOf(service, fa.Name.Element.Name) == enum {
				return expr, true
			}
			return nil, false
		}
	}
	return v.enumConversion(expr, service, enum), true
}
