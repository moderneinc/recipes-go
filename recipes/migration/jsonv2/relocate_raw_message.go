/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Rewrites json.RawMessage type references and conversions to jsontext.Value and
// swaps the import, unless a v1 symbol v2 removed would be stranded.
type RelocateRawMessage struct {
	recipe.Base
	preserveV1 bool
}

func (r *RelocateRawMessage) Name() string {
	return "org.openrewrite.golang.migration.RelocateRawMessage"
}
func (r *RelocateRawMessage) DisplayName() string {
	return "Relocate `json.RawMessage` to `jsontext.Value`"
}
func (r *RelocateRawMessage) Description() string {
	return "Rewrite `json.RawMessage` to `jsontext.Value`, swapping the import to `encoding/json/v2`."
}
func (r *RelocateRawMessage) Tags() []string { return []string{"migration", "json"} }

func (r *RelocateRawMessage) Editor() recipe.TreeVisitor {
	return visitor.Init(&relocateRawMessageVisitor{preserveV1: r.preserveV1})
}

type relocateRawMessageVisitor struct {
	visitor.GoVisitor
	preserveV1 bool
	jsonPkg    string
}

func (v *relocateRawMessageVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	jsonPkg := regularJsonPackage(cu)
	// The added jsontext import is emitted under its default name, so a file that
	// already binds `jsontext` is skipped to avoid a colliding reference.
	if jsonPkg == "" || jsonPkg == "jsontext" {
		return cu
	}
	if importsEncodingJsonV2(cu) {
		return cu
	}
	// A time.Duration or fixed [N]byte field encodes incompatibly under bare v2,
	// so the default path leaves the file for review; the compat path migrates it,
	// since PreserveV1Semantics restores the v1 encoding.
	if !v.preserveV1 && fileNeedsV1Compat(cu) {
		return cu
	}

	scan := visitor.Init(&rawMessageBlockerScan{jsonPkg: jsonPkg})
	scan.Visit(cu, nil)
	if scan.blocked || !scan.rawMessageSeen {
		return cu
	}

	v.jsonPkg = jsonPkg
	if scan.jsonReferencedAfter {
		// A surviving Marshal/Unmarshal keeps encoding/json referenced, so it is
		// renamed to the v2 path.
		queueImportSwapToV2(v)
	} else {
		// RawMessage is the only usage and becomes jsontext.Value, so the
		// encoding/json import would be left unused and is dropped instead.
		v.DoAfterVisit((&recipegolang.RemoveImport{PackagePath: "encoding/json"}).Editor())
	}

	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)
	return drainQueuedImports(v, cu, p)
}

// Rewrites a json.RawMessage type reference to jsontext.Value, keeping the
// original whitespace.
func (v *relocateRawMessageVisitor) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)
	target, ok := fa.Target.(*java.Identifier)
	if !ok || target.Name != v.jsonPkg || fa.Name.Element == nil || fa.Name.Element.Name != "RawMessage" {
		return fa
	}
	recipegolang.MaybeAddImport(v, "encoding/json/jsontext", nil, false)
	c := *fa
	c.Target = &java.Identifier{Prefix: target.Prefix, Name: "jsontext"}
	c.Name = java.LeftPadded[*java.Identifier]{
		Before:  fa.Name.Before,
		Element: &java.Identifier{Prefix: fa.Name.Element.Prefix, Name: "Value"},
	}
	c.Type = nil
	return &c
}

// Rewrites a json.RawMessage(x) conversion to jsontext.Value(x).
func (v *relocateRawMessageVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.Select == nil || selectIdentName(mi) != v.jsonPkg || mi.Name == nil || mi.Name.Name != "RawMessage" {
		return mi
	}
	sel, ok := mi.Select.Element.(*java.Identifier)
	if !ok {
		return mi
	}
	recipegolang.MaybeAddImport(v, "encoding/json/jsontext", nil, false)
	c := *mi
	c.Select = &java.RightPadded[java.Expression]{
		Element: &java.Identifier{Prefix: sel.Prefix, Name: "jsontext"},
		After:   mi.Select.After,
	}
	c.Name = &java.Identifier{Prefix: mi.Name.Prefix, Name: "Value"}
	return &c
}

type rawMessageBlockerScan struct {
	visitor.GoVisitor
	jsonPkg             string
	blocked             bool
	rawMessageSeen      bool
	jsonReferencedAfter bool
}

func (s *rawMessageBlockerScan) VisitIdentifier(id *java.Identifier, p any) java.J {
	// The migration introduces a jsontext reference, so any existing binding of
	// that name would collide with or shadow the added import.
	if id.Name == "jsontext" {
		s.blocked = true
	}
	return s.GoVisitor.VisitIdentifier(id, p)
}

func (s *rawMessageBlockerScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if s.blocked {
		return mi
	}
	if mi.Name != nil && selectIdentName(mi) == s.jsonPkg {
		name := mi.Name.Name
		if name == "RawMessage" {
			s.rawMessageSeen = true
			return s.GoVisitor.VisitMethodInvocation(mi, p)
		}
		if !isV2SurvivingJsonName(name) {
			s.blocked = true
			return mi
		}
		// A surviving Marshal/Unmarshal keeps encoding/json referenced after the
		// RawMessage rewrite.
		s.jsonReferencedAfter = true
		return s.GoVisitor.VisitMethodInvocation(mi, p)
	}
	// A method call on a v1 Encoder/Decoder value, which v2 relocated to jsontext.
	if declaring := declaringFQN(mi); declaring == "encoding/json" || strings.HasPrefix(declaring, "encoding/json.") {
		s.blocked = true
		return mi
	}
	return s.GoVisitor.VisitMethodInvocation(mi, p)
}

func (s *rawMessageBlockerScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if s.blocked {
		return fa
	}
	if ident, ok := fa.Target.(*java.Identifier); ok && ident.Name == s.jsonPkg {
		nm := fa.Name.Element
		if nm != nil && nm.Name == "RawMessage" {
			s.rawMessageSeen = true
			return s.GoVisitor.VisitFieldAccess(fa, p)
		}
		// Encoder, Decoder, Number, an error type, and every other removed type
		// cannot survive the import swap.
		if nm == nil || !isV2SurvivingJsonName(nm.Name) {
			s.blocked = true
			return fa
		}
		// A surviving Marshaler/Unmarshaler keeps encoding/json referenced.
		s.jsonReferencedAfter = true
	}
	return s.GoVisitor.VisitFieldAccess(fa, p)
}
