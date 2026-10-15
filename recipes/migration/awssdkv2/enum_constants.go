/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// v1's enum constants were untyped strings, usable anywhere a string was; v2's
// are typed, usable only where the enum is. The migration types v2's enum
// fields, so a constant headed for one keeps the type. Everywhere else — a map
// of strings, a comparison against a field read as a string, an argument — it
// is converted back to the string v1 code used it as.

// enumSlot reports whether the enum constant at cursor lands in a slot v2 types
// as its enum: a shape field, an element of an enum list, a setter or an
// assignment to one. v1 wrapped the constant in aws.String for any of them.
func (v *migrateVisitor) enumSlot(cursor *visitor.Cursor) bool {
	type ancestor struct {
		tree   java.Tree
		cursor *visitor.Cursor
	}
	var up []ancestor
	for c := cursor.Parent(); c != nil && len(up) < 6; c = c.Parent() {
		if c.Value() != nil {
			up = append(up, ancestor{c.Value(), c})
		}
	}
	node := cursor.Value()
	i := 0
	at := func() java.Tree {
		if i < len(up) {
			return up[i].tree
		}
		return nil
	}
	if call, isCall := at().(*java.MethodInvocation); isCall {
		if helper, isAws := qualifiedCall(call, v.scan.awsPkg); isAws && helper == "String" {
			node, i = call, i+1
		}
	}
	// An element of a list literal: the list is what lands in the field.
	listElement := false
	if comp, isComposite := at().(*golang.Composite); isComposite {
		node, i, listElement = comp, i+1, true
		if call, isCall := at().(*java.MethodInvocation); isCall {
			if helper, isAws := qualifiedCall(call, v.scan.awsPkg); isAws && helper == "StringSlice" {
				node, i = call, i+1
			}
		}
	}
	switch parent := at().(type) {
	case *golang.KeyValue:
		key, isIdent := parent.Key.(*java.Identifier)
		if !isIdent || parent.Value.Element != node || i+1 >= len(up) {
			return false
		}
		comp, isComposite := up[i+1].tree.(*golang.Composite)
		if !isComposite {
			return false
		}
		service, shape, ok := v.scan.compositeShapeAt(comp, up[i+1].cursor)
		if !ok {
			return false
		}
		if listElement {
			return enumSliceOf(service, shape, key.Name) != ""
		}
		return awsmanifest.FieldEnum(service, shape, key.Name) != ""
	case *java.Assignment:
		if listElement || parent.Value.Element != node {
			return false
		}
		service, shape, field, ok := v.assignedShapeField(parent.Variable)
		return ok && awsmanifest.FieldEnum(service, shape, field) != ""
	case *java.MethodInvocation:
		if listElement {
			return false
		}
		owner, field, _, ok := v.scan.shapeSetterTarget(parent)
		return ok && awsmanifest.FieldEnum(owner.service, owner.shape, field) != ""
	}
	return false
}
