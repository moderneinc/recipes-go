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

// v2 made some v1 shapes unions: an interface each of whose member types holds
// one of the v1 shape's fields. A v1 literal setting exactly one field is that
// member, holding the field's shape by value, as in4it/ecs-deploy's
// appmesh.ServiceDiscovery setting only Dns becomes ServiceDiscoveryMemberDns.
// A literal setting none or several fields, or setting one to anything but a
// literal of its shape, has no single member to become, and holds the file
// back.

// unionLiteral reports whether comp is a v1 union literal setting one field to
// a literal of its shape, returning the member type and the value it holds.
func (s *fileScan) unionLiteral(comp *golang.Composite) (service, member string, value *golang.Composite, ok bool) {
	service, shape, isShape := s.compositeShape(comp)
	if !isShape || !awsmanifest.IsUnion(service, shape) {
		return "", "", nil, false
	}
	var kvs []*golang.KeyValue
	for _, rp := range comp.Elements.Elements {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		kv, isKeyValue := rp.Element.(*golang.KeyValue)
		if !isKeyValue {
			return "", "", nil, false
		}
		kvs = append(kvs, kv)
	}
	if len(kvs) != 1 {
		return "", "", nil, false
	}
	key, isIdent := kvs[0].Key.(*java.Identifier)
	if !isIdent {
		return "", "", nil, false
	}
	member, isMember := awsmanifest.UnionMember(service, shape, key.Name)
	if !isMember {
		return "", "", nil, false
	}
	unary, isUnary := kvs[0].Value.Element.(*golang.Unary)
	if !isUnary || unary.Operator.Element != golang.AddressOf {
		return "", "", nil, false
	}
	value, isLiteral := unary.Expression.(*golang.Composite)
	if !isLiteral {
		return "", "", nil, false
	}
	return service, member, value, true
}

// unionMemberLiteral builds the member literal holding the visited value.
func (v *migrateVisitor) unionMemberLiteral(comp *golang.Composite, service, member string) *golang.Composite {
	var value java.Expression
	for _, rp := range comp.Elements.Elements {
		if kv, isKeyValue := rp.Element.(*golang.KeyValue); isKeyValue {
			if unary, isUnary := kv.Value.Element.(*golang.Unary); isUnary {
				value = unary.Expression
			}
		}
	}
	alias := awsmanifest.TypesAlias(service)
	v.needsTypes[service] = true
	v.typesAliases[alias] = true
	c := *comp
	c.TypeExpr = &java.FieldAccess{
		Prefix: comp.TypeExpr.GetPrefix(),
		Target: &java.Identifier{Name: alias, Type: lstutil.NamedType(v2ServicePath(service) + "/types")},
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: member}},
		Type:   lstutil.NamedType(v2ServicePath(service) + "/types." + member),
	}
	kv := comp.Elements.Elements
	for i, rp := range kv {
		if old, isKeyValue := rp.Element.(*golang.KeyValue); isKeyValue {
			replaced := *old
			replaced.Key = &java.Identifier{Prefix: old.Key.GetPrefix(), Name: "Value"}
			replaced.Value = java.LeftPadded[java.Expression]{Before: old.Value.Before, Element: lstutil.SetExprPrefix(value, old.Value.Element.GetPrefix()), Markers: old.Value.Markers}
			elements := make([]java.RightPadded[java.Expression], len(kv))
			copy(elements, kv)
			elements[i].Element = &replaced
			c.Elements.Elements = elements
			break
		}
	}
	return &c
}

// unionType reports whether fa, already relocated to the types package, names
// a union.
func (v *migrateVisitor) unionType(fa *java.FieldAccess) bool {
	target, isIdent := fa.Target.(*java.Identifier)
	if !isIdent || fa.Name.Element == nil {
		return false
	}
	for _, service := range v.scan.services {
		if target.Name == awsmanifest.TypesAlias(service) && awsmanifest.IsUnion(service, fa.Name.Element.Name) {
			return true
		}
	}
	return false
}
