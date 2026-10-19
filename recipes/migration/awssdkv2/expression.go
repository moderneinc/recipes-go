/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// v2 moved the DynamoDB expression builder into its own feature module under the
// same package name, keeping the builder API. What changed is underneath: a
// value is marshalled by v2's encoder, which stores an empty string or a
// collection as itself where v1 stored NULL, and Names and Values return v2's
// map types.
const (
	v1DynamoExpression = v1ServicePkg + "dynamodb/expression"
	v2DynamoExpression = v2Module + "/feature/dynamodb/expression"
)

// scalarKinds are the operand types both encoders marshal alike, short of an
// empty string.
var scalarKinds = map[string]bool{
	"string": true, "bool": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
}

// expressionBlockReason names an expression-builder use whose v2 form would
// compile but mean something else, or "" when the call carries over.
func (s *fileScan) expressionBlockReason(mi *java.MethodInvocation, cursor *visitor.Cursor) string {
	if s.expressionPkg == "" || mi.Name == nil {
		return ""
	}
	if call, ok := qualifiedCall(mi, s.expressionPkg); ok {
		operand := -1
		switch call {
		case "Value":
			operand = 0
		case "Contains":
			operand = 1
		}
		if args := realArgs(mi); operand >= 0 && operand < len(args) && !marshalsAlike(args[operand]) {
			return "expression." + call + " of a value v2's encoder may store differently"
		}
		return ""
	}
	// An input field takes v2's maps as readily as it took v1's; anything else
	// done with them is done to a different type.
	if (mi.Name.Name == "Names" || mi.Name.Name == "Values") && len(realArgs(mi)) == 0 && !feedsInputField(cursor) {
		return "expression " + mi.Name.Name + "() used outside an input field, whose type v2 changed"
	}
	return ""
}

// expressionMap reports whether value is an expression's Names or Values, which
// v2's builder already hands back in the form v2's input fields hold.
func (s *fileScan) expressionMap(value java.Expression) bool {
	mi, isCall := value.(*java.MethodInvocation)
	return isCall && s.expressionPkg != "" && mi.Name != nil && mi.Select != nil &&
		(mi.Name.Name == "Names" || mi.Name.Name == "Values") && len(realArgs(mi)) == 0
}

// marshalsAlike reports whether both encoders store the operand the same way: a
// scalar, short of an empty string literal.
func marshalsAlike(arg java.Expression) bool {
	if lit, isLiteral := arg.(*java.Literal); isLiteral {
		return lit.Source != `""` && lit.Source != "``"
	}
	return scalarKinds[matcher.GetFullyQualifiedName(matcher.TypeOfExpression(arg))]
}

// feedsInputField reports whether the call at cursor is the value of a field in
// a composite literal.
func feedsInputField(cursor *visitor.Cursor) bool {
	for c := cursor.Parent(); c != nil; c = c.Parent() {
		if c.Value() == nil {
			continue
		}
		_, isField := c.Value().(*golang.KeyValue)
		return isField
	}
	return false
}
