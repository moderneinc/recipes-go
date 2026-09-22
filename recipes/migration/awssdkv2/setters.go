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
	recv, isIdent := selectIdentifier(mi)
	if !isIdent {
		return shapeRef{}, "", nil, false
	}
	owner, isShape := s.shapes[recv.Name]
	if !isShape {
		return shapeRef{}, "", nil, false
	}
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
	target := &java.FieldAccess{
		Prefix: mi.Prefix,
		Target: lstutil.SetExprPrefix(mi.Select.Element, java.EmptySpace),
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: field}},
	}

	var assigned java.Expression
	switch {
	case awsmanifest.Depointered(owner.service, owner.shape, field) != "":
		// v2 holds the field as the value the setter already took.
		assigned = value
	case awsmanifest.FieldEnum(owner.service, owner.shape, field) != "":
		assigned = v.enumConversion(value, owner.service, awsmanifest.FieldEnum(owner.service, owner.shape, field))
	default:
		helper := awsHelperFor(awsmanifest.PointerScalar(owner.service, owner.shape, field))
		if helper == "" {
			return nil, false
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
