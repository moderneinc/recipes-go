/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"fmt"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/awssdkv2/awsexportdata"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// v1 reported a missing region through a sentinel error; v2 returns a typed one.
// A comparison against the sentinel therefore becomes an errors.As, which is
// what the v2 SDK's own documentation reaches for.
const errMissingRegion = "ErrMissingRegion"

var awsErr = template.Expr("awsErr")

var (
	missingRegionTemplate = template.ExpressionTemplate(
		fmt.Sprintf(`errors.As(%s, new(*aws.MissingRegionError))`, awsErr),
	).Captures(awsErr).Imports(errorsPkg, v2Aws).ExportData(awsexportdata.FS).Build()

	notMissingRegionTemplate = template.ExpressionTemplate(
		fmt.Sprintf(`!errors.As(%s, new(*aws.MissingRegionError))`, awsErr),
	).Captures(awsErr).Imports(errorsPkg, v2Aws).ExportData(awsexportdata.FS).Build()
)

// missingRegionComparison reports whether bin compares an error against v1's
// sentinel, returning the error expression and whether the test was for
// inequality.
func (s *fileScan) missingRegionComparison(bin *java.Binary) (errExpr java.Expression, negated, ok bool) {
	switch bin.Operator.Element {
	case java.Equal:
	case java.NotEqual:
		negated = true
	default:
		return nil, false, false
	}
	if s.isMissingRegionSentinel(bin.Right) {
		return bin.Left, negated, true
	}
	if s.isMissingRegionSentinel(bin.Left) {
		return bin.Right, negated, true
	}
	return nil, false, false
}

func (s *fileScan) isMissingRegionSentinel(expr java.Expression) bool {
	fa, isField := expr.(*java.FieldAccess)
	if !isField {
		return false
	}
	name, isAws := qualifiedRef(fa, s.awsPkg)
	return isAws && name == errMissingRegion
}

// missingRegionScan counts the references to the sentinel against the ones a
// comparison accounts for, so a reference the rewrite would leave behind — one
// stored in a variable, say — holds the file back instead.
type missingRegionScan struct {
	visitor.GoVisitor
	scan       *fileScan
	references int
	compared   int
}

func (v *missingRegionScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if v.scan.isMissingRegionSentinel(fa) {
		v.references++
	}
	return v.GoVisitor.VisitFieldAccess(fa, p)
}

func (v *missingRegionScan) VisitBinary(bin *java.Binary, p any) java.J {
	if _, _, ok := v.scan.missingRegionComparison(bin); ok {
		v.compared++
	}
	return v.GoVisitor.VisitBinary(bin, p)
}

// missingRegionRewrite turns the comparison into the errors.As v2 wants.
func (v *migrateVisitor) missingRegionRewrite(bin *java.Binary) (java.J, bool) {
	errExpr, negated, ok := v.scan.missingRegionComparison(bin)
	if !ok {
		return nil, false
	}
	t := missingRegionTemplate
	if negated {
		t = notMissingRegionTemplate
	}
	applied := t.Apply(v.Cursor(), template.NewMatchResult().Bind(awsErr, lstutil.SetExprPrefix(errExpr, java.EmptySpace)))
	if applied == nil {
		return nil, false
	}
	v.needsErrors = true
	v.needsAws = true
	v.awsUsed = true
	return lstutil.SetExprPrefix(applied.(java.Expression), bin.Prefix), true
}
