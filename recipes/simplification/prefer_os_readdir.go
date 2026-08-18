/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"fmt"

	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/template"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var rdirName = template.Expr("rdirName")

// PreferOsReadDir replaces `ioutil.ReadDir(name)` with `os.ReadDir(name)` (Go 1.16+).
// Staticcheck: SA1019 (deprecated)
type PreferOsReadDir struct {
	recipe.Base
}

func (r *PreferOsReadDir) Name() string {
	return "org.openrewrite.golang.codequality.PreferOsReadDir"
}
func (r *PreferOsReadDir) DisplayName() string {
	return "Prefer os.ReadDir"
}
func (r *PreferOsReadDir) Description() string {
	return "Replace deprecated `ioutil.ReadDir(name)` with `os.ReadDir(name)` (Go 1.16+)."
}
func (r *PreferOsReadDir) Tags() []string { return []string{"cleanup", "simplification"} }

func (r *PreferOsReadDir) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

var (
	readDirPattern = template.Expression(fmt.Sprintf(`ioutil.ReadDir(%s)`, rdirName)).
			Captures(rdirName).Imports("io/ioutil").Build()
	readDirTemplate = template.ExpressionTemplate(fmt.Sprintf(`os.ReadDir(%s)`, rdirName)).
			Captures(rdirName).Imports("os").Build()
)

func (r *PreferOsReadDir) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferOsReadDirVisitor{})
}

type preferOsReadDirVisitor struct {
	visitor.GoVisitor
}

func (v *preferOsReadDirVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	match := readDirPattern.Match(mi, nil)
	if match == nil {
		return mi
	}

	// Skip when the result is required as []os.FileInfo by a return or a typed
	// variable declaration, where os.ReadDir's []os.DirEntry would not compile.
	if t, ok := requiredResultType(v.Cursor()); ok && t == "[]os.FileInfo" {
		return mi
	}

	replaced := readDirTemplate.Apply(nil, match)
	if replaced == nil {
		return mi
	}
	newCall, ok := replaced.(*java.MethodInvocation)
	if !ok {
		return mi
	}

	recipegolang.MaybeAddImport(v, "os", nil, false)
	recipegolang.MaybeRemoveImport(v, "io/ioutil")
	return newCall.WithPrefix(mi.GetPrefix())
}
