/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"strings"

	"github.com/google/uuid"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Relocates a function-local json.Encoder/Decoder to jsontext and rewrites its
// Encode/Decode calls to the v2 package functions, applied only when every such
// value stays local and no v1 symbol v2 removed would be stranded.
type RelocateEncoderDecoderTypes struct {
	recipe.Base
	preserveV1 bool
}

func (r *RelocateEncoderDecoderTypes) Name() string {
	return "org.openrewrite.golang.migration.RelocateEncoderDecoderTypes"
}
func (r *RelocateEncoderDecoderTypes) DisplayName() string {
	return "Relocate `json.Encoder` and `json.Decoder` to `jsontext`"
}
func (r *RelocateEncoderDecoderTypes) Description() string {
	return "Relocate function-local `json.Encoder`/`json.Decoder` values to `jsontext` and rewrite their `enc.Encode(v)`/`dec.Decode(&v)` calls to `json.MarshalEncode(enc, v)`/`json.UnmarshalDecode(dec, &v)`, swapping the import to `encoding/json/v2`."
}
func (r *RelocateEncoderDecoderTypes) Tags() []string { return []string{"migration", "json"} }

func (r *RelocateEncoderDecoderTypes) Editor() recipe.TreeVisitor {
	return visitor.Init(&relocateEncoderDecoderVisitor{preserveV1: r.preserveV1})
}

type relocateEncoderDecoderVisitor struct {
	visitor.GoVisitor
	preserveV1    bool
	jsonPkg       string
	localEncoders map[string]bool
}

func (v *relocateEncoderDecoderVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	jsonPkg := regularJsonPackage(cu)
	if jsonPkg == "" {
		return cu
	}
	// The added jsontext import is emitted under its default name, so a file that
	// already binds `jsontext` (as the json alias or anything else) is skipped to
	// avoid a colliding or shadowed reference.
	if jsonPkg == "jsontext" {
		return cu
	}
	if importsEncodingJsonV2(cu) {
		return cu
	}
	// A time.Duration field marshals to a runtime error under v2 without an
	// explicit format, so the file is left for review; the compat path skips this
	// guard, since PreserveV1Semantics restores the v1 representation.
	if !v.preserveV1 && fileHasDurationField(cu) {
		return cu
	}

	locals, safe := analyzeRelocation(cu, jsonPkg)
	if !safe {
		return cu
	}

	v.jsonPkg = jsonPkg
	v.localEncoders = locals
	queueImportSwapToV2(v)

	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *relocateEncoderDecoderVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if r, ok := v.relocateConstructor(mi); ok {
		return r
	}
	if r, ok := v.relocateValueCall(mi); ok {
		return r
	}
	return mi
}

// Rewrites json.NewEncoder/NewDecoder to the jsontext constructor, keeping the
// receiver's prefix and arguments.
func (v *relocateEncoderDecoderVisitor) relocateConstructor(mi *java.MethodInvocation) (*java.MethodInvocation, bool) {
	if !isJsonNewCodec(mi, v.jsonPkg) {
		return nil, false
	}
	recv := mi.Select.Element.(*java.Identifier)
	recipegolang.MaybeAddImport(v, "encoding/json/jsontext", nil, false)
	c := *mi
	c.Select = &java.RightPadded[java.Expression]{
		Element: &java.Identifier{Prefix: recv.Prefix, Name: "jsontext"},
		After:   mi.Select.After,
	}
	return &c, true
}

// Rewrites enc.Encode(v) to json.MarshalEncode(enc, v) and dec.Decode(&v) to
// json.UnmarshalDecode(dec, &v) for a local encoder/decoder.
func (v *relocateEncoderDecoderVisitor) relocateValueCall(mi *java.MethodInvocation) (*java.MethodInvocation, bool) {
	fn, ok := encoderValueCallTarget(mi)
	if !ok || mi.Select == nil {
		return nil, false
	}
	recv, ok := mi.Select.Element.(*java.Identifier)
	if !ok || !v.localEncoders[recv.Name] || len(mi.Arguments.Elements) != 1 {
		return nil, false
	}
	value := mi.Arguments.Elements[0]
	return &java.MethodInvocation{
		Prefix: mi.Prefix,
		Select: &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: v.jsonPkg}},
		Name:   &java.Identifier{Name: fn},
		Arguments: java.Container[java.Expression]{
			Elements: []java.RightPadded[java.Expression]{
				{Element: withExprWhitespace(recv, ""), After: java.EmptySpace},
				{Element: withLeadingSpace(value.Element), After: value.After},
			},
		},
	}, true
}

// Reports whether cu can be fully and safely relocated, and returns the names
// of the local encoder/decoder variables.
func analyzeRelocation(cu *golang.CompilationUnit, jsonPkg string) (map[string]bool, bool) {
	collect := visitor.Init(&relocateCollect{
		jsonPkg: jsonPkg,
		locals:  map[string]bool{},
		ctorIDs: map[uuid.UUID]struct{}{},
	})
	collect.Visit(cu, nil)
	if len(collect.locals) == 0 {
		return nil, false
	}

	verify := visitor.Init(&relocateVerify{
		jsonPkg: jsonPkg,
		locals:  collect.locals,
		ctorIDs: collect.ctorIDs,
	})
	verify.Visit(cu, nil)
	if verify.blocked || !verify.jsonReferenced {
		return nil, false
	}

	// Escape check, scoped per function so a codec name may repeat across
	// functions: a local encoder must appear exactly once as its declaration plus
	// once per Encode/Decode call, and any extra occurrence means the value
	// escaped by being passed, returned, stored, or taken as a method value.
	escape := visitor.Init(&relocateEscape{jsonPkg: jsonPkg})
	escape.Visit(cu, nil)
	if escape.blocked {
		return nil, false
	}
	return collect.locals, true
}

type relocateEscape struct {
	visitor.GoVisitor
	jsonPkg string
	blocked bool
}

func (e *relocateEscape) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	if md.Body != nil && !functionEncodersDoNotEscape(md.Body, e.jsonPkg) {
		e.blocked = true
	}
	return e.GoVisitor.VisitMethodDeclaration(md, p)
}

// Reports whether every local encoder/decoder declared in a single function
// body is used only as an Encode/Decode receiver, by counting occurrences
// within that body.
func functionEncodersDoNotEscape(body *java.Block, jsonPkg string) bool {
	collect := visitor.Init(&relocateCollect{
		jsonPkg: jsonPkg,
		locals:  map[string]bool{},
		ctorIDs: map[uuid.UUID]struct{}{},
	})
	collect.Visit(body, nil)
	if len(collect.locals) == 0 {
		return true
	}
	counter := visitor.Init(&relocateBodyCounter{
		jsonPkg:     jsonPkg,
		locals:      collect.locals,
		occurrences: map[string]int{},
		encodeCalls: map[string]int{},
	})
	counter.Visit(body, nil)
	for name := range collect.locals {
		if counter.occurrences[name] != 1+counter.encodeCalls[name] {
			return false
		}
	}
	return true
}

type relocateBodyCounter struct {
	visitor.GoVisitor
	jsonPkg     string
	locals      map[string]bool
	occurrences map[string]int
	encodeCalls map[string]int
}

func (c *relocateBodyCounter) VisitIdentifier(id *java.Identifier, p any) java.J {
	if c.locals[id.Name] {
		c.occurrences[id.Name]++
	}
	return c.GoVisitor.VisitIdentifier(id, p)
}

func (c *relocateBodyCounter) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if _, ok := encoderValueCallTarget(mi); ok {
		if recv, ok := mi.Select.Element.(*java.Identifier); ok && c.locals[recv.Name] {
			c.encodeCalls[recv.Name]++
		}
	}
	return c.GoVisitor.VisitMethodInvocation(mi, p)
}

type relocateCollect struct {
	visitor.GoVisitor
	jsonPkg string
	locals  map[string]bool
	ctorIDs map[uuid.UUID]struct{}
}

func (c *relocateCollect) VisitAssignment(assign *java.Assignment, p any) java.J {
	assign = c.GoVisitor.VisitAssignment(assign, p).(*java.Assignment)
	if java.HasMarker[golang.ShortVarDecl](assign.Markers) {
		if id, ok := assign.Variable.(*java.Identifier); ok {
			if mi, ok := assign.Value.Element.(*java.MethodInvocation); ok && isJsonNewCodec(mi, c.jsonPkg) {
				c.locals[id.Name] = true
				c.ctorIDs[mi.ID] = struct{}{}
			}
		}
	}
	return assign
}

func (c *relocateCollect) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	vd = c.GoVisitor.VisitVariableDeclarations(vd, p).(*java.VariableDeclarations)
	if vd.TypeExpr != nil {
		return vd
	}
	for _, decl := range vd.Variables {
		d := decl.Element
		if d == nil || d.Name == nil || d.Initializer == nil {
			continue
		}
		if mi, ok := d.Initializer.Element.(*java.MethodInvocation); ok && isJsonNewCodec(mi, c.jsonPkg) {
			c.locals[d.Name.Name] = true
			c.ctorIDs[mi.ID] = struct{}{}
		}
	}
	return vd
}

type relocateVerify struct {
	visitor.GoVisitor
	jsonPkg        string
	locals         map[string]bool
	ctorIDs        map[uuid.UUID]struct{}
	blocked        bool
	jsonReferenced bool
}

func (s *relocateVerify) VisitIdentifier(id *java.Identifier, p any) java.J {
	// The migration introduces a jsontext reference, so any existing binding of
	// that name would collide with or shadow the added import.
	if id.Name == "jsontext" {
		s.blocked = true
	}
	return s.GoVisitor.VisitIdentifier(id, p)
}

func (s *relocateVerify) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if s.blocked {
		return mi
	}
	// A json.NewEncoder/NewDecoder call is fine only when it initializes a local
	// variable; otherwise its result escapes.
	if selectIdentName(mi) == s.jsonPkg && mi.Name != nil && (mi.Name.Name == "NewEncoder" || mi.Name.Name == "NewDecoder") {
		if _, ok := s.ctorIDs[mi.ID]; !ok {
			s.blocked = true
			return mi
		}
		return s.GoVisitor.VisitMethodInvocation(mi, p)
	}
	// A surviving json package function (Marshal, Unmarshal) stays as is and
	// adopts v2 semantics; any other one v2 dropped and this recipe cannot
	// rewrite it.
	if selectIdentName(mi) == s.jsonPkg {
		if mi.Name == nil || !isV2SurvivingJsonName(mi.Name.Name) {
			s.blocked = true
			return mi
		}
		return s.GoVisitor.VisitMethodInvocation(mi, p)
	}
	// An Encode/Decode call is fine only on a local encoder/decoder identifier.
	if _, ok := encoderValueCallTarget(mi); ok {
		recv, ok := mi.Select.Element.(*java.Identifier)
		if !ok || !s.locals[recv.Name] {
			s.blocked = true
			return mi
		}
		s.jsonReferenced = true
		return s.GoVisitor.VisitMethodInvocation(mi, p)
	}
	// Any other method on a json Encoder/Decoder is out of scope.
	if declaring := declaringFQN(mi); declaring == "encoding/json.Encoder" || declaring == "encoding/json.Decoder" ||
		declaring == "encoding/json" || strings.HasPrefix(declaring, "encoding/json.") {
		s.blocked = true
		return mi
	}
	return s.GoVisitor.VisitMethodInvocation(mi, p)
}

func (s *relocateVerify) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if s.blocked {
		return fa
	}
	// A json.<Name> reference to a type v2 removed (Encoder, Decoder, RawMessage,
	// Number, an error type) cannot survive the import swap.
	if jsonFieldAccessBlocks(fa, s.jsonPkg) {
		s.blocked = true
		return fa
	}
	return s.GoVisitor.VisitFieldAccess(fa, p)
}

// Reports whether mi is a json.NewEncoder or json.NewDecoder call on the given
// local json package name.
func isJsonNewCodec(mi *java.MethodInvocation, jsonPkg string) bool {
	if selectIdentName(mi) != jsonPkg || mi.Name == nil {
		return false
	}
	if _, ok := mi.Select.Element.(*java.Identifier); !ok {
		return false
	}
	return mi.Name.Name == "NewEncoder" || mi.Name.Name == "NewDecoder"
}

// Returns the v2 package function for an enc.Encode or dec.Decode call resolved
// to encoding/json, or false when mi is neither.
func encoderValueCallTarget(mi *java.MethodInvocation) (string, bool) {
	if mi.Name == nil {
		return "", false
	}
	switch declaringFQN(mi) {
	case "encoding/json.Encoder":
		if mi.Name.Name == "Encode" {
			return "MarshalEncode", true
		}
	case "encoding/json.Decoder":
		if mi.Name.Name == "Decode" {
			return "UnmarshalDecode", true
		}
	}
	return "", false
}
