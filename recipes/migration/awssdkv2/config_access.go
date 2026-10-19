/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"strconv"
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"

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
	// offset is what v2's value adds to v1's. v1 counted retries and v2 counts
	// attempts, so v1's n is v2's n+1 — which also carries v1's -1, the
	// service default, to v2's 0, which means the same.
	offset int64
}

// configTargets are the settings with a v2 counterpart under the same reach.
// Anything else v1's Config carried is configured somewhere else entirely in v2.
var configTargets = map[string]configTarget{
	"Region": {name: "Region", depointered: true},
	// v2 calls it a CredentialsProvider, and the constructors that fill it are
	// renamed or stood in for where they are.
	"Credentials": {name: "Credentials"},
	"MaxRetries":  {name: "RetryMaxAttempts", depointered: true, offset: 1},
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
	if holder, isIdent := outer.Target.(*java.Identifier); isIdent && (s.configRestructured[holder.Name] || s.configKept[holder.Name]) {
		return holder, false, target, true
	}
	inner, isField := outer.Target.(*java.FieldAccess)
	if !isField || inner.Name.Element == nil || inner.Name.Element.Name != configField {
		// A config value the file keeps holds the settings itself.
		if s.configValue(outer.Target) {
			return outer.Target, false, target, true
		}
		return nil, false, configTarget{}, false
	}
	// A session is as often a struct's field as a variable.
	if s.sessionHolder(inner.Target) {
		return inner.Target, false, target, true
	}
	holder, isIdent := inner.Target.(*java.Identifier)
	if !isIdent {
		return nil, false, configTarget{}, false
	}
	_, isPackage := s.services[holder.Name]
	if s.clients[holder.Name] == "" || isPackage {
		return nil, false, configTarget{}, false
	}
	return holder, true, target, true
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
			MethodType: lstutil.FuncType(v2ServicePath(v.scan.clients[identifierName(holder)]), "Options", nil),
		}
	}
	return &java.FieldAccess{
		Prefix: expr.GetPrefix(),
		Target: lstutil.SetExprPrefix(receiver, java.EmptySpace),
		Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: target.name}},
	}, target, true
}

// configValue reports whether expr holds a v1 config the file keeps as v2's: an
// element of a `...*aws.Config` parameter, a local a client takes, or a value
// attribution types as one.
func (s *fileScan) configValue(expr java.Expression) bool {
	switch e := expr.(type) {
	case *java.ArrayAccess:
		if id, isIdent := e.Indexed.(*java.Identifier); isIdent && s.configVariadics[id.Name] {
			return true
		}
	case *java.Identifier:
		if s.clientConfigs[e.Name] {
			return true
		}
	}
	fqn := matcher.GetFullyQualifiedName(matcher.TypeOfExpression(expr))
	return fqn == v1Aws+".Config"
}

// sessionHolder reports whether expr names a session: a variable, or a field,
// declared as one.
func (s *fileScan) sessionHolder(expr java.Expression) bool {
	switch e := expr.(type) {
	case *java.Identifier:
		return s.sessionVars[e.Name]
	case *java.FieldAccess:
		return e.Name.Element != nil && s.sessionVars[e.Name.Element.Name]
	}
	return false
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
	if !v.scan.sessionHolder(fa.Target) {
		holder, isIdent := fa.Target.(*java.Identifier)
		if !isIdent {
			return fa
		}
		if _, isPackage := v.scan.services[holder.Name]; v.scan.clients[holder.Name] == "" || isPackage {
			return fa
		}
	}
	// The reach is only usable as `<holder>.Config.<setting>`; the Config value
	// itself has no v2 counterpart to hand around.
	parent, isSetting := v.Cursor().Parent().Value().(*java.FieldAccess)
	// A setter called on a session's config for its effect is an assignment.
	if call, isCall := v.Cursor().Parent().Value().(*java.MethodInvocation); isCall && v.scan.sessionHolder(fa.Target) {
		if _, _, ok := v.scan.sessionConfigSetter(call); ok {
			return fa
		}
	}
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
	} else if target.offset != 0 && dereferenced(v.Cursor().Parent()) && writtenThrough(v.Cursor().Parent().Parent()) {
		// The read becomes v2's value less the offset, which cannot be written to.
		v.scan.configAccessBlocked = "Config." + target.name + " written through the pointer v1 held it in"
	}
	return fa
}

// writtenThrough reports whether the dereference the cursor points at is
// written to: assigned, compound-assigned, incremented or decremented.
func writtenThrough(cursor *visitor.Cursor) bool {
	switch parent := parentTree(cursor).(type) {
	case *java.Assignment:
		return parent.Variable == cursor.Value()
	case *java.AssignmentOperation, *golang.AssignmentOperation:
		return true
	case *java.Unary:
		switch parent.Operator.Element {
		case java.PostIncrement, java.PostDecrement:
			return true
		}
	}
	return false
}

// shifted adds offset to a setting's value, folding it into an integer literal.
func shifted(expr java.Expression, offset int64) java.Expression {
	if offset == 0 {
		return expr
	}
	value, isConstant := intConstant(expr)
	if isConstant {
		v := value + offset
		return &java.Literal{Prefix: expr.GetPrefix(), Value: v, Source: strconv.FormatInt(v, 10), Type: lstutil.NamedType("int")}
	}
	op, amount := java.Add, offset
	if offset < 0 {
		op, amount = java.Subtract, -offset
	}
	// Every Go integer operator binds at least as tightly as + and -, so the
	// operand needs no parentheses.
	return &java.Binary{
		Prefix:   expr.GetPrefix(),
		Left:     lstutil.SetExprPrefix(expr, java.EmptySpace),
		Operator: java.LeftPadded[java.BinaryOperator]{Before: java.SingleSpace, Element: op},
		Right:    &java.Literal{Prefix: java.SingleSpace, Value: amount, Source: strconv.FormatInt(amount, 10), Type: lstutil.NamedType("int")},
		Type:     lstutil.NamedType("int"),
	}
}

// intConstant reads an integer literal, negated or not.
func intConstant(expr java.Expression) (int64, bool) {
	sign := int64(1)
	if unary, isUnary := expr.(*java.Unary); isUnary && unary.Operator.Element == java.Negate {
		sign, expr = -1, unary.Operand
	}
	lit, isLiteral := expr.(*java.Literal)
	if !isLiteral {
		return 0, false
	}
	value, isInt := lit.Value.(int64)
	return sign * value, isInt
}

func parenthesized(expr java.Expression) java.Expression {
	return &java.Parentheses{
		Prefix: expr.GetPrefix(),
		Tree:   java.RightPadded[java.Expression]{Element: lstutil.SetExprPrefix(expr, java.EmptySpace)},
	}
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
		// A local only a client takes keeps its literals, retyped where they stand.
		_, isConfigLocal := scan.configLocals[target.Name]
		return isConfigLocal && !scan.clientConfigs[target.Name] && !scan.configKept[target.Name]
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
		if key.Name == retryerField {
			if retryer, ok := v.retryerValue(kv.Value.Element); ok {
				replaced := *kv
				replaced.Value = java.LeftPadded[java.Expression]{Before: kv.Value.Before, Element: lstutil.SetExprPrefix(retryer, kv.Value.Element.GetPrefix()), Markers: kv.Value.Markers}
				elements[i].Element = &replaced
				changed = true
			}
			continue
		}
		if key.Name == s3ForcePathStyle {
			force, unwrapped := v.scan.pointerValueSource(kv.Value.Element)
			if !unwrapped {
				continue
			}
			replaced := *kv
			replaced.Key = &java.Identifier{Prefix: key.Prefix, Name: "ServiceOptions"}
			replaced.Value = java.LeftPadded[java.Expression]{Before: kv.Value.Before, Element: lstutil.SetExprPrefix(serviceOptionsLiteral(pathStyleCall(force)), kv.Value.Element.GetPrefix()), Markers: kv.Value.Markers}
			elements[i].Element = &replaced
			changed = true
			v.needsPathStyle = true
			continue
		}
		target, mapped := configTargets[key.Name]
		if !mapped {
			continue
		}
		value := kv.Value.Element
		// The endpoint takes the scheme v1 would have added, inside the pointer
		// v2 still holds it by.
		if key.Name == "Endpoint" {
			if call, isCall := value.(*java.MethodInvocation); isCall {
				if helper, isAws := qualifiedCall(call, v.scan.awsPkg); isAws && helper == "String" && len(realArgs(call)) == 1 {
					c := *call
					c.Arguments.Elements = []java.RightPadded[java.Expression]{{Element: endpointCall(realArgs(call)[0])}}
					value = &c
					v.needsConfigHelper = true
				}
			}
		}
		if target.depointered {
			inner, unwrapped := v.scan.pointerValueSource(value)
			if !unwrapped {
				continue
			}
			value = lstutil.SetExprPrefix(shifted(inner, target.offset), value.GetPrefix())
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

// v1's Credentials answered Get with a Value naming the key, the secret and the
// session token, as v2's provider answers Retrieve; it takes the context v1's
// GetWithContext did.
var credentialsRetrieves = map[string]bool{"Get": true, "GetWithContext": true}

// configCredentialsCall reports whether mi calls Get or GetWithContext on a
// config's Credentials, returning the context it takes, or nil for the
// enclosing function's.
func (s *fileScan) configCredentialsCall(mi *java.MethodInvocation) (fa *java.FieldAccess, ctx java.Expression, ok bool) {
	if mi.Name == nil || mi.Select == nil {
		return nil, nil, false
	}
	fa, isField := mi.Select.Element.(*java.FieldAccess)
	if !isField {
		return nil, nil, false
	}
	if _, _, target, isConfig := s.configAccess(fa); !isConfig || target.name != credentialsField {
		return nil, nil, false
	}
	args := realArgs(mi)
	switch {
	case mi.Name.Name == "Get" && len(args) == 0:
		return fa, nil, true
	case mi.Name.Name == "GetWithContext" && len(args) == 1:
		return fa, args[0], true
	}
	return fa, nil, false
}

// v1 named the provider that supplied a credentials Value; v2's Credentials
// names its Source instead, some sources spelled differently.
const providerNameField = "ProviderName"

// credentialValueScan records the names bound to credentials a config's
// provider answered.
type credentialValueScan struct {
	visitor.GoVisitor
	scan *fileScan
}

func (v *credentialValueScan) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	if len(ma.Values) == 1 && len(ma.Variables) == 2 {
		if call, isCall := ma.Values[0].Element.(*java.MethodInvocation); isCall {
			if _, _, ok := v.scan.configCredentialsCall(call); ok {
				if id, isIdent := ma.Variables[0].Element.(*java.Identifier); isIdent {
					v.scan.credentialValues[id.Name] = true
				}
			}
		}
	}
	return v.GoVisitor.VisitMultiAssignment(ma, p)
}

// providerNameRead reports whether fa reads a credentials Value's provider name.
func (s *fileScan) providerNameRead(fa *java.FieldAccess) bool {
	if fa.Name.Element == nil || fa.Name.Element.Name != providerNameField {
		return false
	}
	if id, isIdent := fa.Target.(*java.Identifier); isIdent && s.credentialValues[id.Name] {
		return true
	}
	return matcher.GetFullyQualifiedName(matcher.TypeOfExpression(fa.Target)) == v1Credentials+".Value"
}

// sessionConfigSetter reads `<session>.Config.WithX(v)`, returning the v2
// setting and the value. Called for its effect, as a statement, it set the
// session's config in place, which in v2 is the config itself.
func (s *fileScan) sessionConfigSetter(mi *java.MethodInvocation) (configTarget, java.Expression, bool) {
	if mi.Name == nil || mi.Select == nil || len(realArgs(mi)) != 1 {
		return configTarget{}, nil, false
	}
	field, isSetter := strings.CutPrefix(mi.Name.Name, "With")
	target, mapped := configTargets[field]
	if !isSetter || !mapped {
		return configTarget{}, nil, false
	}
	fa, isField := mi.Select.Element.(*java.FieldAccess)
	if !isField || fa.Name.Element == nil || fa.Name.Element.Name != configField || !s.sessionHolder(fa.Target) {
		return configTarget{}, nil, false
	}
	return target, realArgs(mi)[0], true
}

// sessionConfigAssignment rewrites the setter statement as the assignment.
func (v *migrateVisitor) sessionConfigAssignment(mi *java.MethodInvocation) (java.Statement, bool) {
	target, value, ok := v.scan.sessionConfigSetter(mi)
	if !ok {
		return nil, false
	}
	holder := mi.Select.Element.(*java.FieldAccess).Target
	switch target.name {
	case "BaseEndpoint":
		v.needsConfigHelper, v.awsUsed, v.needsAws = true, true, true
		value = &java.MethodInvocation{
			Select:     &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: v.scan.awsQualifier(), Type: lstutil.NamedType(v2Aws)}},
			Name:       &java.Identifier{Name: "String"},
			Arguments:  java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{{Element: endpointCall(lstutil.SetExprPrefix(value, java.EmptySpace))}}},
			MethodType: lstutil.FuncType(v2Aws, "String", nil),
		}
	default:
		value = shifted(value, target.offset)
	}
	return &java.Assignment{
		Prefix: mi.Prefix,
		Variable: &java.FieldAccess{
			Target: lstutil.SetExprPrefix(holder, java.EmptySpace),
			Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: target.name}},
		},
		Value: java.LeftPadded[java.Expression]{Before: java.SingleSpace, Element: lstutil.SetExprPrefix(value, java.SingleSpace)},
	}, true
}
