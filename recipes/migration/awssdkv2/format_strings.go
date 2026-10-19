/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"go/version"
	"path"
	"strings"

	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// printfCheckGo is the release from which vet reports a printf call whose
// format is not a constant and which has no other arguments.
const printfCheckGo = "1.24"

// raisesPastPrintfCheck reports whether the migration raises the go directive
// of the module holding sourcePath from below the printf check to at or above
// it, which turns code that vetted clean into a vet failure.
func (a *migrateAcc) raisesPastPrintfCheck(sourcePath string) bool {
	a.usage.resolve()
	if !a.usage.migrated {
		return false
	}
	dir, best := path.Dir(sourcePath), ""
	found := false
	for moduleDir := range a.usage.modules {
		if (moduleDir == "." || dir == moduleDir || strings.HasPrefix(dir, moduleDir+"/")) && (!found || len(moduleDir) > len(best)) {
			best, found = moduleDir, true
		}
	}
	if !found {
		return false
	}
	goVersion := a.usage.modules[best].goVersion
	return goVersion != "" && version.Compare("go"+goVersion, "go"+printfCheckGo) < 0
}

// fixFormatStrings passes each non-constant format with nothing to format
// through "%s", as vet's own suggested fix does. The output is unchanged but
// for a format holding a verb, which printed as `%!d(MISSING)` before.
func fixFormatStrings(t java.J) java.J {
	if fixed, ok := visitor.Init(&formatStringVisitor{}).Visit(t, nil).(java.J); ok {
		return fixed
	}
	return t
}

type formatStringVisitor struct {
	visitor.GoVisitor
}

func (v *formatStringVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	at, ok := printfFormatIndex(mi.MethodType)
	if !ok || len(mi.Arguments.Elements) != at+1 {
		return mi
	}
	format := mi.Arguments.Elements[at]
	if !nonConstant(format.Element) {
		return mi
	}
	verb := &java.Literal{Prefix: format.Element.GetPrefix(), Value: "%s", Source: `"%s"`, Type: lstutil.StringType}
	elements := make([]java.RightPadded[java.Expression], 0, at+2)
	elements = append(elements, mi.Arguments.Elements[:at]...)
	elements = append(elements,
		java.RightPadded[java.Expression]{Element: verb},
		java.RightPadded[java.Expression]{Element: lstutil.SetExprPrefix(format.Element, java.SingleSpace), After: format.After, Markers: format.Markers},
	)
	c := *mi
	c.Arguments.Elements = elements
	return &c
}

// printfFormatIndex returns where a printf-style function takes its format:
// a string named format, followed by nothing but `...any`. vet infers a
// wrapper from what it forwards to, whatever it is called, as dregsy's
// raiseError is; the parameter names stand in for that here.
func printfFormatIndex(mt *java.JavaTypeMethod) (int, bool) {
	if mt == nil || len(mt.ParameterTypes) < 2 || len(mt.ParameterNames) != len(mt.ParameterTypes) {
		return 0, false
	}
	n := len(mt.ParameterTypes)
	if mt.ParameterNames[n-2] != "format" || !matcher.IsString(mt.ParameterTypes[n-2]) {
		return 0, false
	}
	args, isSlice := mt.ParameterTypes[n-1].(*java.JavaTypeArray)
	if !isSlice {
		return 0, false
	}
	elem, isClass := args.ElemType.(*java.JavaTypeClass)
	if !isClass || elem.Kind != "Interface" || len(elem.Methods) > 0 {
		return 0, false
	}
	return n - 2, true
}

// nonConstant reports whether e certainly is not a constant expression, so
// vet would report it. Anything it cannot tell is left alone.
func nonConstant(e java.Expression) bool {
	switch n := e.(type) {
	case *java.Identifier:
		return n.FieldType != nil
	case *java.FieldAccess:
		return n.Name.Element != nil && n.Name.Element.FieldType != nil
	case *java.MethodInvocation, *java.ArrayAccess:
		return true
	case *java.Binary:
		return nonConstant(n.Left) || nonConstant(n.Right)
	case *java.Parentheses:
		return nonConstant(n.Tree.Element)
	case *java.TypeCast:
		return nonConstant(n.Expr)
	}
	return false
}
