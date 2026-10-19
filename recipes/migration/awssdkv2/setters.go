/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"strings"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

// v1 gave every shape a fluent setter taking the plain value and storing the
// pointer; v2 dropped them all. The assignment that replaces one has to put the
// value into whatever v2 holds the field in, which is not the same answer for
// every field: a pointer scalar takes the aws helper the setter applied, an
// enum takes a conversion, and a field v2 holds by value takes the value as it
// stands.
const setterPrefix = "Set"

// shapeSetterTarget names the field a fluent setter writes, when the receiver's
// shape is known and the field is one the rewrite can assign to.
func (s *fileScan) shapeSetterTarget(mi *java.MethodInvocation) (owner shapeRef, field string, value java.Expression, ok bool) {
	if mi.Name == nil || !strings.HasPrefix(mi.Name.Name, setterPrefix) || mi.Name.Name == setterPrefix {
		return shapeRef{}, "", nil, false
	}
	if mi.Select == nil {
		return shapeRef{}, "", nil, false
	}
	// The receiver may be a field, or an element of a list, as well as a local.
	owner, isShape := s.receiverShape(mi.Select.Element)
	if !isShape {
		return shapeRef{}, "", nil, false
	}
	owner.shape = awsmanifest.V1ShapeName(owner.service, owner.shape)
	args := realArgs(mi)
	if len(args) != 1 {
		return shapeRef{}, "", nil, false
	}
	field = strings.TrimPrefix(mi.Name.Name, setterPrefix)
	if !s.assignableField(owner, field) {
		return shapeRef{}, "", nil, false
	}
	return owner, field, args[0], true
}

// assignableField reports whether the rewrite knows what v2 holds the field in.
func (s *fileScan) assignableField(owner shapeRef, field string) bool {
	switch {
	case awsmanifest.Depointered(owner.service, owner.shape, field) != "":
		return true
	case awsmanifest.FieldEnum(owner.service, owner.shape, field) != "":
		return !awsmanifest.IsValueSlice(owner.service, owner.shape, field)
	case awsmanifest.PointerScalar(owner.service, owner.shape, field) != "":
		return true
	// Any other field's setter took what an assignment to the field takes, a
	// list, a map or a nested shape, which converts as one does.
	case awsmanifest.IsValueSlice(owner.service, owner.shape, field),
		awsmanifest.IsMapOfValues(owner.service, owner.shape, field),
		awsmanifest.IsMapOfValueSlices(owner.service, owner.shape, field),
		awsmanifest.FieldShape(owner.service, owner.shape, field) != "":
		return true
	}
	return false
}

// setterAssignment rewrites `x.SetF(v)` as `x.F = …`, wrapping the value the
// way v2 holds the field.
func (v *migrateVisitor) setterAssignment(mi *java.MethodInvocation) (java.Statement, bool) {
	owner, field, value, ok := v.scan.shapeSetterTarget(mi)
	if !ok {
		return nil, false
	}
	// An element of a list v2 holds by value is written in v2's list itself,
	// not in the copy the read conversion hands back.
	receiver := mi.Select.Element
	if index, isIndex := receiver.(*java.ArrayAccess); isIndex {
		if call, wrapped := index.Indexed.(*java.MethodInvocation); wrapped && call.Name != nil && call.Name.Name == compatPtrs && len(realArgs(call)) == 1 {
			c := *index
			c.Indexed = lstutil.SetExprPrefix(realArgs(call)[0], index.Indexed.GetPrefix())
			receiver = &c
		}
	}
	target := &java.FieldAccess{
		Prefix: mi.Prefix,
		Target: lstutil.SetExprPrefix(receiver, java.EmptySpace),
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: field}},
	}

	var assigned java.Expression
	switch {
	case awsmanifest.Depointered(owner.service, owner.shape, field) != "":
		// v2 holds the field as the value the setter already took.
		assigned = value
		if from := awsmanifest.NarrowedFrom(owner.service, owner.shape, field); from != "" {
			assigned = narrowedValue(value, from, awsmanifest.Depointered(owner.service, owner.shape, field))
		}
	case awsmanifest.FieldEnum(owner.service, owner.shape, field) != "":
		assigned = v.enumConversion(value, owner.service, awsmanifest.FieldEnum(owner.service, owner.shape, field))
	case awsmanifest.PointerScalar(owner.service, owner.shape, field) == "":
		converted, ok := v.convertedFieldValue(owner.service, owner.shape, field, value)
		if !ok {
			// A nested shape v2 holds as v1 did.
			if awsmanifest.FieldShape(owner.service, owner.shape, field) == "" {
				return nil, false
			}
			converted = value
		}
		assigned = converted
	default:
		helper := awsHelperFor(awsmanifest.PointerScalar(owner.service, owner.shape, field))
		if helper == "" {
			return nil, false
		}
		// The setter took v1's width; v2's helper takes its own.
		if from, to, changed := awsmanifest.ScalarChange(owner.service, owner.shape, field); changed {
			value = narrowedValue(value, from, to)
		}
		v.awsUsed = true
		v.needsAws = true
		assigned = &java.MethodInvocation{
			Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: v.scan.awsQualifier(), Type: lstutil.NamedType(v2Aws)}},
			Name:       &java.Identifier{Name: helper},
			Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(value, java.EmptySpace)}}},
			MethodType: lstutil.FuncType(v2Aws, helper, nil),
		}
	}
	return &java.Assignment{
		Prefix:   mi.Prefix,
		Variable: lstutil.SetExprPrefix(target, java.EmptySpace),
		Value:    java.LeftPadded[java.Expression]{Before: java.SingleSpace, Element: lstutil.SetExprPrefix(assigned, java.SingleSpace)},
	}, true
}
