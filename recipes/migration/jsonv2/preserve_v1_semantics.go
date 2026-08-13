/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// The alias used for the v1 encoding/json import, matching the Go docs, since a
// migrated file already binds json to encoding/json/v2.
const jsonV1Alias = "jsonv1"

// Appends jsonv1.DefaultOptionsV1() to encoding/json/v2 marshal and unmarshal
// calls so their output matches encoding/json v1 byte for byte.
type PreserveV1Semantics struct {
	recipe.Base
}

func (r *PreserveV1Semantics) Name() string {
	return "org.openrewrite.golang.migration.PreserveV1Semantics"
}
func (r *PreserveV1Semantics) DisplayName() string {
	return "Preserve v1 semantics on `encoding/json/v2` calls"
}
func (r *PreserveV1Semantics) Description() string {
	return "Append `jsonv1.DefaultOptionsV1()` to `encoding/json/v2` marshal and unmarshal calls (`Marshal`, `Unmarshal`, `MarshalWrite`, `UnmarshalRead`, `MarshalEncode`, `UnmarshalDecode`) so their output matches v1 byte for byte, adding the `jsonv1 \"encoding/json\"` import. `DefaultOptionsV1` is the v1 compatibility bundle from the `encoding/json` package and re-enables every v1 default that v2 changed. Runs only on files that import `encoding/json/v2`, and skips a call that already passes `DefaultOptionsV1`."
}
func (r *PreserveV1Semantics) Tags() []string { return []string{"migration", "json"} }

func (r *PreserveV1Semantics) Editor() recipe.TreeVisitor {
	return visitor.Init(&preserveV1SemanticsVisitor{})
}

type preserveV1SemanticsVisitor struct {
	visitor.GoVisitor
	jsonPkg string
}

func (v *preserveV1SemanticsVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.jsonPkg = localJsonV2Package(cu)
	if v.jsonPkg == "" || v.jsonPkg == "_" || v.jsonPkg == "." {
		return cu
	}
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *preserveV1SemanticsVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if selectIdentName(mi) != v.jsonPkg || mi.Name == nil || !takesV1Options(mi.Name.Name) {
		return mi
	}
	if hasDefaultOptionsV1(mi) {
		return mi
	}
	alias := jsonV1Alias
	recipegolang.MaybeAddImport(v, "encoding/json", &alias, false)
	elems := append([]java.RightPadded[java.Expression]{}, mi.Arguments.Elements...)
	elems = append(elems, java.RightPadded[java.Expression]{
		Element: defaultOptionsV1Call(),
		After:   java.EmptySpace,
	})
	c := *mi
	c.Arguments.Elements = elems
	return &c
}

// Reports whether the v2 package function accepts trailing Options that
// DefaultOptionsV1 can set.
func takesV1Options(name string) bool {
	switch name {
	case "Marshal", "Unmarshal", "MarshalWrite", "UnmarshalRead", "MarshalEncode", "UnmarshalDecode":
		return true
	}
	return false
}

// Reports whether the call already passes a DefaultOptionsV1() argument.
func hasDefaultOptionsV1(mi *java.MethodInvocation) bool {
	for _, e := range mi.Arguments.Elements {
		if opt, ok := e.Element.(*java.MethodInvocation); ok && opt.Name != nil && opt.Name.Name == "DefaultOptionsV1" {
			return true
		}
	}
	return false
}

// Builds a jsonv1.DefaultOptionsV1() call spaced to follow a comma.
func defaultOptionsV1Call() *java.MethodInvocation {
	return &java.MethodInvocation{
		Prefix: java.SingleSpace,
		Select: &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: jsonV1Alias}},
		Name:   &java.Identifier{Name: "DefaultOptionsV1"},
	}
}

// Returns the local name a file uses for encoding/json/v2 (its alias, or json),
// or an empty string when the package is not imported.
func localJsonV2Package(cu *golang.CompilationUnit) string {
	if cu.Imports == nil {
		return ""
	}
	for _, imp := range cu.Imports.Elements {
		if path, ok := importPath(imp.Element); ok && path == "encoding/json/v2" {
			if alias := importAlias(imp.Element); alias != "" {
				return alias
			}
			return "json"
		}
	}
	return ""
}
