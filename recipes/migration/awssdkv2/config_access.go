/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// v1 let a caller reach the settings a session or a client was built with
// through a `Config` field. v2 has no such field: the session's config is the
// value that replaced it, and a client answers with `Options()`. The settings
// themselves mostly kept their names, though the ones v1 held as pointers so
// that "unset" was expressible are plain values in v2.
const configField = "Config"

type configTarget struct {
	// name is what v2 calls the setting.
	name string
	// depointered records that v2 holds it by value where v1 held a pointer, so
	// a read of it loses its dereference and a write its aws helper.
	depointered bool
}

// configTargets are the settings with a v2 counterpart under the same reach.
// Anything else v1's Config carried is configured somewhere else entirely in v2.
var configTargets = map[string]configTarget{
	"Region": {name: "Region", depointered: true},
	// v2 calls it a CredentialsProvider, and the constructors that fill it are
	// renamed or stood in for where they are.
	"Credentials": {name: "Credentials"},
	"MaxRetries":  {name: "RetryMaxAttempts", depointered: true},
	"Endpoint":    {name: "BaseEndpoint"},
	"HTTPClient":  {name: "HTTPClient"},
}

// configAccess reports whether expr reads `<session or client>.Config.<field>`,
// returning the receiver, the setting v2 calls it, and whether the dereference
// around it goes.
func (s *fileScan) configAccess(expr java.Expression) (receiver java.Expression, viaClient bool, target configTarget, ok bool) {
	outer, isField := expr.(*java.FieldAccess)
	if !isField || outer.Name.Element == nil {
		return nil, false, configTarget{}, false
	}
	target, mapped := configTargets[outer.Name.Element.Name]
	if !mapped {
		return nil, false, configTarget{}, false
	}

	// A config local is the config itself, so the setting sits one level up
	// from where a session or a client kept it.
	if holder, isIdent := outer.Target.(*java.Identifier); isIdent && s.configRestructured[holder.Name] {
		return holder, false, target, true
	}

	inner, isField := outer.Target.(*java.FieldAccess)
	if !isField || inner.Name.Element == nil || inner.Name.Element.Name != configField {
		return nil, false, configTarget{}, false
	}
	holder, isIdent := inner.Target.(*java.Identifier)
	if !isIdent {
		return nil, false, configTarget{}, false
	}
	_, isPackage := s.services[holder.Name]
	switch {
	case s.sessionVars[holder.Name]:
	case s.clients[holder.Name] != "" && !isPackage:
		viaClient = true
	default:
		return nil, false, configTarget{}, false
	}
	return holder, viaClient, target, true
}

// rewriteConfigAccess builds the v2 reach for the setting: straight off the
// config value that replaced the session, or through the client's Options.
func (v *migrateVisitor) rewriteConfigAccess(expr java.Expression) (java.Expression, configTarget, bool) {
	holder, viaClient, target, ok := v.scan.configAccess(expr)
	if !ok {
		return nil, configTarget{}, false
	}
	receiver := holder
	if viaClient {
		receiver = &java.MethodInvocation{
			Select:     &java.RightPadded[java.Expression]{Element: lstutil.SetExprPrefix(holder, java.EmptySpace)},
			Name:       &java.Identifier{Name: "Options"},
			Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: &java.Empty{}}}},
			MethodType: lstutil.FuncType(v2ServicePkg+v.scan.clients[identifierName(holder)], "Options", nil),
		}
	}
	return &java.FieldAccess{
		Prefix: expr.GetPrefix(),
		Target: lstutil.SetExprPrefix(receiver, java.EmptySpace),
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: target.name}},
	}, target, true
}

func identifierName(expr java.Expression) string {
	if id, isIdent := expr.(*java.Identifier); isIdent {
		return id.Name
	}
	return ""
}

// configAccessScan checks every reach through a Config field against what v2
// leaves in its place: a setting with no counterpart, or one v2 holds by value
// read without a dereference, holds the file back.
type configAccessScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *configAccessScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)
	if fa.Name.Element == nil || fa.Name.Element.Name != configField {
		return fa
	}
	holder, isIdent := fa.Target.(*java.Identifier)
	if !isIdent {
		return fa
	}
	_, isPackage := v.scan.services[holder.Name]
	if !v.scan.sessionVars[holder.Name] && (v.scan.clients[holder.Name] == "" || isPackage) {
		return fa
	}
	// The reach is only usable as `<holder>.Config.<setting>`; the Config value
	// itself has no v2 counterpart to hand around.
	parent, isSetting := v.Cursor().Parent().Value().(*java.FieldAccess)
	if !isSetting {
		v.scan.configAccessBlocked = configField + " read off a session or a client"
		return fa
	}
	if _, _, target, ok := v.scan.configAccess(parent); !ok {
		name := ""
		if parent.Name.Element != nil {
			name = parent.Name.Element.Name
		}
		v.scan.configAccessBlocked = "Config." + name + ", which v2 configures elsewhere"
	} else if target.depointered && !dereferenced(v.Cursor().Parent()) && !assignedTo(v.Cursor().Parent()) {
		v.scan.configAccessBlocked = "Config." + target.name + " read as the pointer v1 held it in"
	}
	return fa
}

// assignedTo reports whether the node the cursor points at is the target of an
// assignment, which is a write rather than a read.
func assignedTo(cursor *visitor.Cursor) bool {
	parent := cursor.Parent()
	if parent == nil {
		return false
	}
	assignment, isAssignment := parent.Value().(*java.Assignment)
	return isAssignment && assignment.Variable == cursor.Value()
}

// v1 code often held an aws.Config of its own and handed it to the session when
// it came to build a client. v2's aws.Config is the config, so such a value
// carries over as it stands — but the fields inside it do not all keep their v1
// shape, and a literal the session does not consume has to be brought to the v2
// one where it is written.

// awsConfigLiteral reports whether a composite constructs an aws.Config.
func (s *fileScan) awsConfigLiteral(comp *golang.Composite) bool {
	fa, isField := comp.TypeExpr.(*java.FieldAccess)
	if !isField {
		return false
	}
	name, isAws := qualifiedRef(fa, s.awsPkg)
	return isAws && name == "Config"
}

// consumedAsSessionConfig reports whether the literal is one the rewrite takes
// apart elsewhere: an argument to a session constructor, the Config of a
// session's options, or the initialiser of a config local. Those are replaced
// whole, so retyping their fields in place would be wasted at best and would
// defeat the option reader at worst.
func consumedAsSessionConfig(scan *fileScan, cursor *visitor.Cursor) bool {
	parent := cursor.Parent()
	if parent == nil {
		return false
	}
	if _, isAddressOf := parent.Value().(*golang.Unary); isAddressOf {
		parent = parent.Parent()
		if parent == nil {
			return false
		}
	}
	switch node := parent.Value().(type) {
	case *java.MethodInvocation:
		call, isSession := qualifiedCall(node, scan.sessionPkg)
		return isSession && (call == "NewSession" || call == "New")
	case *golang.KeyValue:
		key, isIdent := node.Key.(*java.Identifier)
		return isIdent && key.Name == sessionConfig
	case *java.Assignment:
		target, isIdent := node.Variable.(*java.Identifier)
		if !isIdent {
			return false
		}
		_, isConfigLocal := scan.configLocals[target.Name]
		return isConfigLocal
	}
	return false
}

// retypedConfigLiteral brings an aws.Config literal to the shape v2 holds its
// settings in — the region by value, the endpoint under its new name.
func (v *migrateVisitor) retypedConfigLiteral(comp *golang.Composite) (java.J, bool) {
	elements := make([]java.RightPadded[java.Expression], len(comp.Elements.Elements))
	copy(elements, comp.Elements.Elements)
	changed := false
	for i, rp := range elements {
		kv, isKeyValue := rp.Element.(*golang.KeyValue)
		if !isKeyValue {
			continue
		}
		key, isIdent := kv.Key.(*java.Identifier)
		if !isIdent {
			continue
		}
		target, mapped := configTargets[key.Name]
		if !mapped {
			continue
		}
		value := kv.Value.Element
		if target.depointered {
			inner, unwrapped := v.scan.pointerValueSource(value)
			if !unwrapped {
				continue
			}
			value = lstutil.SetExprPrefix(inner, value.GetPrefix())
		}
		replaced := *kv
		replaced.Key = &java.Identifier{Prefix: key.Prefix, Name: target.name, Type: key.Type}
		replaced.Value = java.LeftPadded[java.Expression]{Before: kv.Value.Before, Element: value, Markers: kv.Value.Markers}
		elements[i].Element = &replaced
		changed = true
	}
	if !changed {
		return comp, false
	}
	c := *comp
	c.Elements = comp.Elements
	c.Elements.Elements = elements
	v.awsUsed = true
	v.needsAws = true
	return &c, true
}
