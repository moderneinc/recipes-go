/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"strings"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsmanifest"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// v1's DynamoDB AttributeValue was a struct with a pointer field per member,
// one of them set; v2's is an interface with a struct per member. A value
// built with one scalar member set becomes that member's struct, and a read of
// a scalar member goes through a generated getter answering nil where the
// value holds another member — as v1's unset field did. The set, map and list
// members, a member written in place, and a value that may be nil — which v1
// sent as an empty AttributeValue the service rejected — are held back.
const (
	dynamodbService    = "dynamodb"
	attributeValue     = "AttributeValue"
	attributeValueFile = "zz_awssdkv2_attributevalue.go"
)

// attributeMembers maps the scalar members to the basic type v2 holds them as.
var attributeMembers = map[string]string{"S": "string", "N": "string", "B": "[]byte", "BOOL": "bool", "NULL": "bool"}

// attributeGetter names the generated getter for a member.
func attributeGetter(member string) string { return "awsCompatAttribute" + member }

// attributeValueRef reports whether fa names v1's or the relocated
// AttributeValue.
func (s *fileScan) attributeValueRef(fa *java.FieldAccess) bool {
	if fa == nil || fa.Name.Element == nil || fa.Name.Element.Name != attributeValue {
		return false
	}
	target, isIdent := fa.Target.(*java.Identifier)
	if !isIdent {
		return false
	}
	return s.services[target.Name] == dynamodbService || target.Name == awsmanifest.TypesAlias(dynamodbService)
}

// attributeComposite reports whether comp builds an AttributeValue, typed or
// elided inside a list or map of them.
func (s *fileScan) attributeComposite(comp *golang.Composite, cursor *visitor.Cursor) bool {
	service, shape, ok := s.compositeShapeAt(comp, cursor)
	return ok && service == dynamodbService && shape == attributeValue
}

// attributeMember returns the one member an AttributeValue literal sets and the
// value it sets it to.
func attributeMember(comp *golang.Composite) (member string, value java.Expression, ok bool) {
	var elements []*golang.KeyValue
	for _, rp := range comp.Elements.Elements {
		if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
			continue
		}
		kv, isKeyValue := rp.Element.(*golang.KeyValue)
		if !isKeyValue {
			return "", nil, false
		}
		elements = append(elements, kv)
	}
	if len(elements) != 1 {
		return "", nil, false
	}
	key, isIdent := elements[0].Key.(*java.Identifier)
	if !isIdent || attributeMembers[key.Name] == "" {
		return "", nil, false
	}
	return key.Name, elements[0].Value.Element, true
}

// memberValue returns the value v2's member holds for what v1's field was set
// to, when that is provably not nil: the operand of an aws pointer helper or of
// &, a pointer an enclosing if has just tested against nil, or a byte slice
// fresh from io.ReadAll.
func (s *fileScan) memberValue(member string, value java.Expression, cursor *visitor.Cursor) (java.Expression, bool) {
	if attributeMembers[member] == "[]byte" {
		return value, s.freshBytes(value, cursor)
	}
	if inner, ok := s.pointerValueSource(value); ok {
		return inner, true
	}
	key := exprKey(value)
	if key == "" || !nilChecked(cursor, key) {
		return nil, false
	}
	return &golang.Unary{
		Prefix:     value.GetPrefix(),
		Operator:   java.LeftPadded[golang.UnaryOperator]{Element: golang.Indirection},
		Expression: lstutil.SetExprPrefix(value, java.EmptySpace),
	}, true
}

// exprKey spells a name or a chain of field reads, or "" for anything else.
func exprKey(expr java.Expression) string {
	switch e := expr.(type) {
	case *java.Identifier:
		return e.Name
	case *java.FieldAccess:
		if e.Name.Element == nil {
			return ""
		}
		if outer := exprKey(e.Target); outer != "" {
			return outer + "." + e.Name.Element.Name
		}
	}
	return ""
}

// nilChecked reports whether the node at cursor sits in the then-branch of an if
// testing key != nil, with nothing in that branch assigning to key.
func nilChecked(cursor *visitor.Cursor, key string) bool {
	child := cursor.Value()
	for c := cursor.Parent(); c != nil; c = c.Parent() {
		if c.Value() == nil {
			continue
		}
		if iff, isIf := c.Value().(*java.If); isIf && iff.Condition != nil && iff.ThenPart.Element == child {
			if testsNotNil(iff.Condition.Tree.Element, key) && !assigns(iff.ThenPart.Element, key) {
				return true
			}
		}
		child = c.Value()
	}
	return false
}

// testsNotNil reports whether cond is, or conjoins, key != nil.
func testsNotNil(cond java.Expression, key string) bool {
	bin, isBinary := cond.(*java.Binary)
	if !isBinary {
		return false
	}
	switch bin.Operator.Element {
	case java.LogicalAnd:
		return testsNotNil(bin.Left, key) || testsNotNil(bin.Right, key)
	case java.NotEqual:
		nilSide := func(e java.Expression) bool {
			id, isIdent := e.(*java.Identifier)
			return isIdent && id.Name == "nil"
		}
		return (exprKey(bin.Left) == key && nilSide(bin.Right)) || (exprKey(bin.Right) == key && nilSide(bin.Left))
	}
	return false
}

// assigns reports whether anything under tree assigns to key.
func assigns(tree java.Tree, key string) bool {
	scan := visitor.Init(&assignmentScan{key: key})
	scan.Visit(tree, nil)
	return scan.found
}

type assignmentScan struct {
	visitor.GoVisitor
	key   string
	found bool
}

func (v *assignmentScan) VisitAssignment(a *java.Assignment, p any) java.J {
	if exprKey(a.Variable) == v.key {
		v.found = true
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

func (v *assignmentScan) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	for _, rp := range ma.Variables {
		if exprKey(rp.Element) == v.key {
			v.found = true
		}
	}
	return v.GoVisitor.VisitMultiAssignment(ma, p)
}

// freshBytes reports whether value is a byte slice io.ReadAll returned, which
// is never nil: the call itself, or a name the enclosing function bound to its
// result and never reassigned.
func (s *fileScan) freshBytes(value java.Expression, cursor *visitor.Cursor) bool {
	if s.readAll(value) {
		return true
	}
	id, isIdent := value.(*java.Identifier)
	if !isIdent {
		return false
	}
	var fn *java.MethodDeclaration
	for c := cursor.Parent(); c != nil; c = c.Parent() {
		if md, isFunc := c.Value().(*java.MethodDeclaration); isFunc && md.Name != nil && md.Name.Name != "" {
			fn = md
			break
		}
	}
	if fn == nil || fn.Body == nil {
		return false
	}
	scan := visitor.Init(&readAllBindingScan{scan: s, name: id.Name})
	scan.Visit(fn.Body, nil)
	return scan.fromReadAll == 1 && scan.other == 0
}

// readAll reports whether expr calls io.ReadAll or ioutil.ReadAll.
func (s *fileScan) readAll(expr java.Expression) bool {
	mi, isCall := expr.(*java.MethodInvocation)
	if !isCall || mi.Name == nil || mi.Name.Name != "ReadAll" {
		return false
	}
	recv, isIdent := selectIdentifier(mi)
	return isIdent && (recv.Name == s.ioPkg || recv.Name == s.ioutilPkg) && recv.Name != ""
}

type readAllBindingScan struct {
	visitor.GoVisitor
	scan        *fileScan
	name        string
	fromReadAll int
	other       int
}

func (v *readAllBindingScan) VisitMultiAssignment(ma *golang.MultiAssignment, p any) java.J {
	for i, rp := range ma.Variables {
		if id, isIdent := rp.Element.(*java.Identifier); isIdent && id.Name == v.name {
			if i == 0 && len(ma.Values) == 1 && v.scan.readAll(ma.Values[0].Element) {
				v.fromReadAll++
			} else {
				v.other++
			}
		}
	}
	return v.GoVisitor.VisitMultiAssignment(ma, p)
}

func (v *readAllBindingScan) VisitAssignment(a *java.Assignment, p any) java.J {
	if exprKey(a.Variable) == v.name {
		v.other++
	}
	return v.GoVisitor.VisitAssignment(a, p)
}

// attributeRead reports whether fa reads a scalar member of an AttributeValue:
// a member name on a value attribution types as one, or does not type at all.
func (s *fileScan) attributeRead(fa *java.FieldAccess) bool {
	if fa.Name.Element == nil || attributeMembers[fa.Name.Element.Name] == "" || !s.usesService(dynamodbService) {
		return false
	}
	if id, isIdent := fa.Target.(*java.Identifier); isIdent {
		if _, isPackage := s.services[id.Name]; isPackage {
			return false
		}
	}
	fqn := matcher.GetFullyQualifiedName(matcher.TypeOfExpression(fa.Target))
	if fqn == "" {
		return true
	}
	return strings.HasSuffix(fqn, "/service/dynamodb."+attributeValue) || strings.HasSuffix(fqn, "/service/dynamodb/types."+attributeValue)
}

// attributeValueBlockReason names an AttributeValue use the rewrite cannot carry
// over, or "".
func (s *fileScan) attributeValueBlockReason(fa *java.FieldAccess, cursor *visitor.Cursor) string {
	parent := parentTree(cursor)
	switch p := parent.(type) {
	case *golang.PointerType:
		return ""
	case *golang.Composite:
		// A literal's type, which the literal's own check covers, as long as it
		// is a pointer v2's interface can stand in for.
		if p.TypeExpr == java.Expression(fa) {
			if unary, isUnary := parentTree(cursorOf(cursor, p)).(*golang.Unary); isUnary && unary.Operator.Element == golang.AddressOf {
				return ""
			}
		}
	}
	return "an AttributeValue held by value, which v2's interface cannot stand in for"
}

// cursorOf finds the cursor holding tree among cursor's ancestors.
func cursorOf(cursor *visitor.Cursor, tree java.Tree) *visitor.Cursor {
	for c := cursor; c != nil; c = c.Parent() {
		if c.Value() == tree {
			return c
		}
	}
	return cursor
}

// attributeLiteral builds `&dynamodbtypes.AttributeValueMember<X>{Value: v}`,
// or the literal alone where an & already stands in front of it.
func (v *migrateVisitor) attributeLiteral(comp *golang.Composite, addressed bool) (java.Expression, bool) {
	member, value, ok := attributeMember(comp)
	if !ok {
		return nil, false
	}
	inner, ok := v.scan.memberValue(member, value, v.Cursor())
	if !ok {
		return nil, false
	}
	alias := awsmanifest.TypesAlias(dynamodbService)
	v.needsTypes[dynamodbService] = true
	v.typesAliases[alias] = true
	memberType := attributeValue + "Member" + member
	literal := &golang.Composite{
		Prefix: comp.Prefix,
		TypeExpr: &java.FieldAccess{
			Target: &java.Identifier{Name: alias, Type: lstutil.NamedType(v2ServicePkg + dynamodbService + "/types")},
			Name:   java.LeftPadded[*java.Identifier]{Element: &java.Identifier{Name: memberType}},
			Type:   lstutil.NamedType(v2ServicePkg + dynamodbService + "/types." + memberType),
		},
		Elements: comp.Elements,
	}
	literal.Elements.Elements = []java.RightPadded[java.Expression]{{
		Element: &golang.KeyValue{
			Key:   &java.Identifier{Name: "Value"},
			Value: java.LeftPadded[java.Expression]{Before: java.EmptySpace, Element: lstutil.SetExprPrefix(inner, java.SingleSpace)},
		},
	}}
	if addressed {
		return literal, true
	}
	return &golang.Unary{
		Prefix:     comp.Prefix,
		Operator:   java.LeftPadded[golang.UnaryOperator]{Element: golang.AddressOf},
		Expression: lstutil.SetExprPrefix(literal, java.EmptySpace),
	}, true
}

// attributeGetterCall builds the generated getter's call for a member read.
func (v *migrateVisitor) attributeGetterCall(fa *java.FieldAccess) java.Expression {
	getter := attributeGetter(fa.Name.Element.Name)
	v.needsAttributeHelpers = true
	return &java.MethodInvocation{
		Prefix: fa.Prefix,
		Name:   &java.Identifier{Name: getter},
		Arguments: java.Container[java.Expression]{Elements: []java.RightPadded[java.Expression]{
			{Element: lstutil.SetExprPrefix(fa.Target, java.EmptySpace)},
		}},
		MethodType: lstutil.FuncType("", getter, nil),
	}
}

const attributeValueSource = `// Code generated by MigrateAwsSdkGoToV2. DO NOT EDIT.

package %s

import "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

// v1's AttributeValue held a pointer field per member, nil unless that member
// was the one set. v2 made each member a type of its own; these read one back
// the way v1's field did.

func awsCompatAttributeS(av types.AttributeValue) *string {
	if m, ok := av.(*types.AttributeValueMemberS); ok {
		return &m.Value
	}
	return nil
}

func awsCompatAttributeN(av types.AttributeValue) *string {
	if m, ok := av.(*types.AttributeValueMemberN); ok {
		return &m.Value
	}
	return nil
}

func awsCompatAttributeB(av types.AttributeValue) []byte {
	if m, ok := av.(*types.AttributeValueMemberB); ok {
		return m.Value
	}
	return nil
}

func awsCompatAttributeBOOL(av types.AttributeValue) *bool {
	if m, ok := av.(*types.AttributeValueMemberBOOL); ok {
		return &m.Value
	}
	return nil
}

func awsCompatAttributeNULL(av types.AttributeValue) *bool {
	if m, ok := av.(*types.AttributeValueMemberNULL); ok {
		return &m.Value
	}
	return nil
}
`

// ioQualifiers fills the names the standard library's I/O packages are
// imported under.
func (s *fileScan) ioQualifiers(cu *golang.CompilationUnit) {
	s.ioPkg = pathswap.Qualifier(cu, "io")
	s.ioutilPkg = pathswap.Qualifier(cu, "io/ioutil")
	s.bytesPkg = pathswap.Qualifier(cu, "bytes")
	s.stringsPkg = pathswap.Qualifier(cu, "strings")
}
