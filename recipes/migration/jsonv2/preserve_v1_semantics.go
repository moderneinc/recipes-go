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
// calls to re-enable the v1 defaults that v2 changed.
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
	return "Append `jsonv1.DefaultOptionsV1()` to `encoding/json/v2` marshal and unmarshal calls, adding the `jsonv1 \"encoding/json\"` import, to re-enable the v1 defaults that v2 changed. `DefaultOptionsV1` is the v1 compatibility bundle from the `encoding/json` package."
}
func (r *PreserveV1Semantics) Tags() []string { return []string{"migration", "json"} }

func (r *PreserveV1Semantics) Editor() recipe.TreeVisitor {
	return visitor.Init(&preserveV1SemanticsVisitor{})
}

type preserveV1SemanticsVisitor struct {
	visitor.GoVisitor
	jsonPkg     string
	jsontextPkg string
}

func (v *preserveV1SemanticsVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.jsonPkg = localJsonV2Package(cu)
	if v.jsonPkg == "" || v.jsonPkg == "_" || v.jsonPkg == "." {
		return cu
	}
	v.jsontextPkg = localJsontextPackage(cu)
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *preserveV1SemanticsVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.Name == nil {
		return mi
	}
	sel := selectIdentName(mi)
	// For streaming, the options belong on the jsontext codec (which MarshalEncode
	// and UnmarshalDecode read from), not on the marshal call itself; the encoder
	// constructor also carries the syntactic options like HTML escaping.
	if v.jsontextPkg != "" && sel == v.jsontextPkg && (mi.Name.Name == "NewEncoder" || mi.Name.Name == "NewDecoder") {
		return v.appendDefaultOptionsV1(mi)
	}
	// For one-shot calls the options belong on the call, which builds its own codec.
	if sel == v.jsonPkg && takesV1OptionsOnCall(mi.Name.Name) {
		return v.appendDefaultOptionsV1(mi)
	}
	return mi
}

func (v *preserveV1SemanticsVisitor) appendDefaultOptionsV1(mi *java.MethodInvocation) java.J {
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

// Reports whether the v2 package function builds its own codec and so accepts
// trailing Options directly; MarshalEncode/UnmarshalDecode are excluded because
// their options belong on the jsontext codec they are given.
func takesV1OptionsOnCall(name string) bool {
	switch name {
	case "Marshal", "Unmarshal", "MarshalWrite", "UnmarshalRead":
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

// Returns the local name a file uses for encoding/json/jsontext (its alias, or
// jsontext), or an empty string when the package is not imported.
func localJsontextPackage(cu *golang.CompilationUnit) string {
	if cu.Imports == nil {
		return ""
	}
	for _, imp := range cu.Imports.Elements {
		if path, ok := importPath(imp.Element); ok && path == "encoding/json/jsontext" {
			if alias := importAlias(imp.Element); alias != "" {
				return alias
			}
			return "jsontext"
		}
	}
	return ""
}
