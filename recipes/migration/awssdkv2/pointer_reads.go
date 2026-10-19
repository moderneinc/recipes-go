/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

// v1 modelled every optional field as a pointer, so that "unset" had a spelling;
// v2 holds the enums and the plain scalars by value. Code that read one and
// passed it on was written against the pointer, and there is no telling from the
// call site alone what it expected — only that v1 gave it a *T. So the pointer
// is put back at the read, and the code around it is left as it was. A field v1
// could report as nil now reports as a pointer to the zero value, which a
// downstream nil test answers differently; that is inherent to v2 holding the
// field by value.

// pointerRead reports whether fa reads a field v2 holds by value, returning the
// aws helper that restores the pointer and the conversion, if any, that comes
// first. An enum needs one, since its named string type is not a string.
func (s *fileScan) pointerRead(fa *java.FieldAccess) (helper, conversion string, ok bool) {
	if _, isEnum := s.enumFieldOf(fa); isEnum {
		return "String", "string", true
	}
	if basic, isValue := s.depointeredFieldOf(fa); isValue {
		// A narrowed field is widened back to the pointer v1 handed out.
		if from := s.narrowedFieldOf(fa); from != "" {
			return awsHelperFor(from), from, awsHelperFor(from) != ""
		}
		if helper := awsHelperFor(basic); helper != "" {
			return helper, "", true
		}
		if basic == "bool" {
			return "Bool", "", true
		}
	}
	return "", "", false
}

// feedsMatchingField reports whether the read is the value of a field of the
// same kind on another SDK shape, where v2's two ends line up and putting the
// pointer back would break them.
func (s *fileScan) feedsMatchingField(fa *java.FieldAccess, parent, grandparent java.Tree) bool {
	kv, isKeyValue := parent.(*golang.KeyValue)
	if !isKeyValue || kv.Value.Element != java.Expression(fa) {
		return false
	}
	comp, isComposite := grandparent.(*golang.Composite)
	if !isComposite {
		return false
	}
	service, shape, isShape := s.compositeShape(comp)
	if !isShape {
		return false
	}
	key, isIdent := kv.Key.(*java.Identifier)
	if !isIdent {
		return false
	}
	return s.sameFieldKind(fa, service, shape, key.Name)
}

// sameFieldKind reports whether the field being written holds what the field
// being read holds.
func (s *fileScan) sameFieldKind(fa *java.FieldAccess, service, shape, field string) bool {
	if enum, isEnum := s.enumFieldOf(fa); isEnum {
		return awsmanifest.FieldEnum(service, shape, field) == enum
	}
	if basic, isValue := s.depointeredFieldOf(fa); isValue {
		return awsmanifest.Depointered(service, shape, field) == basic
	}
	return false
}

// restorePointer wraps a read in the aws helper that gives back the pointer v1
// handed out.
func (v *migrateVisitor) restorePointer(fa *java.FieldAccess, helper, conversion string) java.Expression {
	inner := java.Expression(lstutil.SetExprPrefix(fa, java.EmptySpace))
	if conversion != "" {
		inner = &java.TypeCast{
			Clazz: &java.ControlParentheses{
				Tree: java.RightPadded[java.Expression]{Element: &java.Identifier{Name: conversion, Type: lstutil.NamedType(conversion)}},
			},
			Expr: inner,
		}
	}
	v.awsUsed = true
	v.needsAws = true
	return &java.MethodInvocation{
		Prefix:     fa.Prefix,
		Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: v.scan.awsQualifier(), Type: lstutil.NamedType(v2Aws)}},
		Name:       &java.Identifier{Name: helper},
		Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: inner}}},
		MethodType: lstutil.FuncType(v2Aws, helper, nil),
	}
}

// awsQualifier is the name the file binds the aws package to, which is the
// package's own where it did not import it before.
func (s *fileScan) awsQualifier() string {
	if s.awsPkg != "" {
		return s.awsPkg
	}
	return "aws"
}

// narrowedValueRead reports whether mi is aws.XValue of a field v2 narrowed,
// returning v1's pointee type, v2's, and whether v2 holds it by value.
func (s *fileScan) narrowedValueRead(mi *java.MethodInvocation) (from, to string, byValue, ok bool) {
	helper, isAws := qualifiedCall(mi, s.awsPkg)
	if !isAws || valueHelperRenames[helper] == "" || len(realArgs(mi)) != 1 {
		return "", "", false, false
	}
	fa, isField := realArgs(mi)[0].(*java.FieldAccess)
	if !isField {
		return "", "", false, false
	}
	return s.narrowedField(fa)
}

// narrowedDeref reports whether unary dereferences a field v2 narrowed but
// still holds by pointer, returning v1's pointee type.
func (s *fileScan) narrowedDeref(unary *golang.Unary) (string, bool) {
	if unary.Operator.Element != golang.Indirection {
		return "", false
	}
	fa, isField := unary.Expression.(*java.FieldAccess)
	if !isField {
		return "", false
	}
	from, _, byValue, ok := s.narrowedField(fa)
	return from, ok && !byValue
}

// narrowedField reports whether fa reads a field v2 narrowed, returning v1's
// pointee type, v2's, and whether v2 holds it by value.
func (s *fileScan) narrowedField(fa *java.FieldAccess) (from, to string, byValue, ok bool) {
	if fa.Name.Element == nil {
		return "", "", false, false
	}
	owner, known := s.receiverShape(fa.Target)
	if !known {
		return "", "", false, false
	}
	// A shape v2 renamed may be known by its v2 name; the manifest keeps v1's.
	shape := awsmanifest.V1ShapeName(owner.service, owner.shape)
	field := fa.Name.Element.Name
	if narrowedFrom := awsmanifest.NarrowedFrom(owner.service, shape, field); narrowedFrom != "" {
		return narrowedFrom, awsmanifest.Depointered(owner.service, shape, field), true, true
	}
	from, to, ok = awsmanifest.ScalarChange(owner.service, shape, field)
	if !ok || awsHelperFor(to) == "" {
		return "", "", false, false
	}
	return from, to, false, true
}

// enumValueRead reports whether mi is aws.StringValue of an enum field.
func (s *fileScan) enumValueRead(mi *java.MethodInvocation) bool {
	helper, isAws := qualifiedCall(mi, s.awsPkg)
	if !isAws || helper != "StringValue" || len(realArgs(mi)) != 1 {
		return false
	}
	fa, isField := realArgs(mi)[0].(*java.FieldAccess)
	if !isField {
		return false
	}
	_, isEnum := s.enumFieldOf(fa)
	return isEnum || s.unresolvedEnumField(fa)
}

// narrowableScalar reports whether a narrowed pointer field's value is one the
// rewrite converts: an aws helper around a value, the address of one, or nil.
func (s *fileScan) narrowableScalar(expr java.Expression) bool {
	if id, isIdent := expr.(*java.Identifier); isIdent && id.Name == "nil" {
		return true
	}
	if unary, isUnary := expr.(*golang.Unary); isUnary && unary.Operator.Element == golang.AddressOf {
		return true
	}
	mi, isCall := expr.(*java.MethodInvocation)
	if !isCall {
		return false
	}
	if _, isNew := newOfValue(mi); isNew {
		return true
	}
	helper, isAws := qualifiedCall(mi, s.awsPkg)
	return isAws && pointerHelpers[helper]
}
