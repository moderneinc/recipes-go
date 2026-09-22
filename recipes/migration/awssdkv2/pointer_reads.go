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
