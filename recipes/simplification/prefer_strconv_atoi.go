/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var atoiS = template.Expr("atoiS")

// PreferStrconvAtoi replaces `strconv.ParseInt(s, 10, 0)` with `strconv.Atoi(s)`.
// Staticcheck: S1030
type PreferStrconvAtoi struct {
	recipe.Base
}

func (r *PreferStrconvAtoi) Name() string {
	return "org.openrewrite.golang.codequality.PreferStrconvAtoi"
}
func (r *PreferStrconvAtoi) DisplayName() string { return "Prefer strconv.Atoi" }
func (r *PreferStrconvAtoi) Description() string {
	return "Replace `strconv.ParseInt(s, 10, 0)` with `strconv.Atoi(s)`."
}
func (r *PreferStrconvAtoi) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferStrconvAtoi) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1030", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var (
	atoiPattern = template.Expression(fmt.Sprintf(`strconv.ParseInt(%s, 10, 0)`, atoiS)).
			Captures(atoiS).Imports("strconv").Build()
	atoiTemplate = template.ExpressionTemplate(fmt.Sprintf(`strconv.Atoi(%s)`, atoiS)).
			Captures(atoiS).Imports("strconv").Build()
)

func (r *PreferStrconvAtoi) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferStrconvAtoiVisitor{})
}

type preferStrconvAtoiVisitor struct {
	visitor.GoVisitor
}

func (v *preferStrconvAtoiVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	match := atoiPattern.Match(mi, nil)
	if match == nil {
		return mi
	}

	// Skip when the int64 result is required as int64 by a return or a typed
	// variable declaration, where strconv.Atoi's int would not compile.
	if t, ok := requiredResultType(v.Cursor()); ok && t == "int64" {
		return mi
	}

	// Skip when a `n, err :=` capture of the int64 result is later returned as int64.
	if capturedValueReturnedAsInt64(v.Cursor()) {
		return mi
	}

	replaced := atoiTemplate.Apply(nil, match)
	if replaced == nil {
		return mi
	}
	newCall, ok := replaced.(*java.MethodInvocation)
	if !ok {
		return mi
	}
	return newCall.WithPrefix(mi.GetPrefix())
}

// Reports whether a `x, err :=` capture of the ParseInt result has x later
// returned at an int64 result position, a best-effort check that misses int64
// parameters, arithmetic, and closures.
func capturedValueReturnedAsInt64(c *visitor.Cursor) bool {
	ma, ok := c.Parent().Value().(*golang.MultiAssignment)
	if !ok || len(ma.Variables) == 0 {
		return false
	}
	// ParseInt's int64 result is captured by the first variable.
	id, ok := ma.Variables[0].Element.(*java.Identifier)
	if !ok || id.Name == "_" {
		return false
	}
	md, ok := visitor.FirstEnclosing[*java.MethodDeclaration](c)
	if !ok {
		return false
	}

	scan := &int64ReturnScanner{varName: id.Name, resultTypes: functionResultTypes(md)}
	scan.Self = scan
	scan.Visit(md, nil)
	return scan.found
}

// Sets found when a `return` yields varName at an int64 result position.
type int64ReturnScanner struct {
	visitor.GoVisitor
	varName     string
	resultTypes []string
	found       bool
}

func (s *int64ReturnScanner) VisitGoReturn(ret *golang.Return, p any) java.J {
	for i, e := range ret.Expressions {
		if id, ok := e.Element.(*java.Identifier); ok && id.Name == s.varName &&
			i < len(s.resultTypes) && s.resultTypes[i] == "int64" {
			s.found = true
		}
	}
	return s.GoVisitor.VisitGoReturn(ret, p)
}

func (s *int64ReturnScanner) VisitReturn(ret *java.Return, p any) java.J {
	if id, ok := ret.Expression.(*java.Identifier); ok && id.Name == s.varName &&
		len(s.resultTypes) >= 1 && s.resultTypes[0] == "int64" {
		s.found = true
	}
	return s.GoVisitor.VisitReturn(ret, p)
}
