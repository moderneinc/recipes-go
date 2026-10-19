/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"strings"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

// github.com/guregu/dynamo is a DynamoDB library built on aws-sdk-go v1, taking
// a session and handing back v1's types; its v2 is built on aws-sdk-go-v2 and
// otherwise keeps its API. A file using it moves to the v2 with the rest: New
// takes the config, every call that reaches DynamoDB takes a context, its
// WithContext forms go, and the limits take an int. What v2 changed the shape
// of — Count's result, an iterator's paging key, the marshalling functions
// over v1's AttributeValue — holds the file back.

const (
	gureguV1 = "github.com/guregu/dynamo"
	gureguV2 = gureguV1 + "/v2"
)

// gureguContextMethods are the calls v2 gives a context.
var gureguContextMethods = map[string]bool{
	"Run": true, "Wait": true, "All": true, "AllWithLastEvaluatedKey": true, "One": true,
	"Value": true, "OldValue": true, "OnlyUpdatedValue": true, "OnlyUpdatedOldValue": true, "Next": true,
}

// gureguIntMethods are the calls whose int64 argument v2 takes as an int.
var gureguIntMethods = map[string]bool{"Limit": true, "SearchLimit": true}

// gureguHeldBack are the names v2 changed the shape of.
var gureguHeldBack = map[string]bool{
	"Count": true, "CountWithContext": true, "LastEvaluatedKey": true, "LastEvaluatedKeys": true,
	"Marshal": true, "Unmarshal": true, "MarshalItem": true, "UnmarshalItem": true,
	"NewFromIface": true, "Segment": true, "AllParallel": true, "IterParallel": true,
	"AllParallelWithLastEvaluatedKeys": true, "MarshalDynamo": true, "UnmarshalDynamo": true,
}

// gureguValue reports whether expr's parse-time type is one of the library's.
func gureguValue(expr java.Expression) bool {
	return strings.HasPrefix(matcher.GetFullyQualifiedName(matcher.TypeOfExpression(expr)), gureguV1+".")
}

// gureguReason names a library call the move to v2 cannot carry.
func (s *fileScan) gureguReason(mi *java.MethodInvocation) string {
	if mi.Name == nil {
		return ""
	}
	name := mi.Name.Name
	if call, isLibrary := qualifiedCall(mi, s.gureguPkg); isLibrary {
		if gureguHeldBack[call] {
			return "dynamo." + call + ", whose v2 takes aws-sdk-go-v2's types"
		}
		return ""
	}
	if mi.Select == nil || !gureguValue(mi.Select.Element) {
		return ""
	}
	if gureguHeldBack[name] || gureguHeldBack[strings.TrimSuffix(name, "WithContext")] {
		return "dynamo " + name + ", whose v2 answers in another shape"
	}
	return ""
}

// gureguCall rewrites a library call as its v2 form.
func (v *migrateVisitor) gureguCall(original, mi *java.MethodInvocation) (java.Expression, bool) {
	if mi.Name == nil {
		return nil, false
	}
	if call, isLibrary := qualifiedCall(mi, v.scan.gureguPkg); isLibrary && call == "New" {
		args := realArgs(mi)
		if len(args) == 0 {
			return nil, false
		}
		// The trailing configs v1 merged over the session's merge over v2's.
		var overrides []java.Expression
		for _, cfg := range args[1:] {
			if !emptyConfigLiteral(cfg) {
				overrides = append(overrides, cfg)
			}
		}
		cfg := args[0]
		if len(overrides) > 0 {
			v.needsConfigHelper = true
			elements := []java.Expression{cfg}
			elements = append(elements, overrides...)
			cfg = helperCallOf(mergeConfigHelper, elements...)
		}
		c := *mi
		c.Arguments.Elements = []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(cfg, java.EmptySpace)}}
		return &c, true
	}
	if original.Select == nil || !gureguValue(original.Select.Element) {
		return nil, false
	}
	name := mi.Name.Name
	if trimmed, withContext := strings.CutSuffix(name, "WithContext"); withContext && gureguContextMethods[trimmed] {
		return renameCall(mi, trimmed, gureguV2).(*java.MethodInvocation), true
	}
	if gureguContextMethods[name] {
		ctx := v.contextExpr()
		if ctx == nil {
			return nil, false
		}
		elements := []java.RightPadded[java.Expression]{{Element: lstutil.SetExprPrefix(ctx, java.EmptySpace)}}
		for _, rp := range mi.Arguments.Elements {
			if _, isEmpty := rp.Element.(*java.Empty); isEmpty {
				continue
			}
			rp.Element = lstutil.SetExprPrefix(rp.Element, java.SingleSpace)
			elements = append(elements, rp)
		}
		c := *mi
		c.Arguments.Elements = elements
		return &c, true
	}
	if gureguIntMethods[name] {
		args := realArgs(mi)
		if len(args) != 1 {
			return nil, false
		}
		if _, isLiteral := args[0].(*java.Literal); isLiteral {
			return nil, false
		}
		c := *mi
		c.Arguments.Elements = []java.RightPadded[java.Expression]{{Element: basicConversion(lstutil.SetExprPrefix(args[0], java.EmptySpace), "int")}}
		return &c, true
	}
	return nil, false
}

// emptyConfigLiteral reports whether expr is `&aws.Config{}`, which merges
// nothing over the session's.
func emptyConfigLiteral(expr java.Expression) bool {
	if unary, isUnary := expr.(*golang.Unary); isUnary {
		expr = unary.Expression
	}
	comp, isComposite := expr.(*golang.Composite)
	if !isComposite {
		return false
	}
	for _, rp := range comp.Elements.Elements {
		if _, isEmpty := rp.Element.(*java.Empty); !isEmpty {
			return false
		}
	}
	return true
}
