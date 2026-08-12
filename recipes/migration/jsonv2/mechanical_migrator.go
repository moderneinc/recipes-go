/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"strings"

	"github.com/google/uuid"

	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// The mechanical encoding/json constructs a recipe may migrate; a file is
// migrated only when every json touchpoint is in this set and json stays
// referenced afterwards.
type mechanicalSet struct {
	streaming     bool // json.NewEncoder(w).Encode(v) / json.NewDecoder(r).Decode(&v)
	marshalIndent bool // json.MarshalIndent(v, prefix, indent)
}

// Classification of a single json touchpoint.
type constructKind int

const (
	constructOther         constructKind = iota // never mechanically migratable
	constructStreaming                          // json.NewEncoder(w).Encode(v) chain
	constructMarshalIndent                      // json.MarshalIndent(...)
)

// Applies the allowed mechanical rewrites and swaps the import, per file, only
// when the file is fully migratable by its allowed set.
type mechanicalMigrator struct {
	visitor.GoVisitor
	allowed mechanicalSet
	jsonPkg string
}

func (v *mechanicalMigrator) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	jsonPkg := localJsonPackage(cu)

	// Only a regular or aliased import is migratable, so skip files that do not
	// import encoding/json and skip blank and dot imports.
	if jsonPkg == "" || jsonPkg == "_" || jsonPkg == "." {
		return cu
	}

	// The rename rewrites encoding/json and every sub-path, so skip files that
	// already reference a sub-path rather than corrupt an encoding/json/jsontext
	// import.
	if importsEncodingJsonSubpath(cu) {
		return cu
	}

	// Leave the file untouched unless every json touchpoint is a construct this
	// recipe handles and json stays referenced afterwards.
	if mechanicalFileBlocked(cu, jsonPkg, v.allowed) {
		return cu
	}

	v.jsonPkg = jsonPkg

	// Queued before the recursion so the rename runs first and never touches a
	// jsontext import added during it.
	v.DoAfterVisit((&recipegolang.RenamePackage{
		OldPackagePath: "encoding/json",
		NewPackagePath: "encoding/json/v2",
	}).Editor())

	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *mechanicalMigrator) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	// A fluent streaming chain is rewritten before recursion so its inner
	// constructor is not visited separately; the rewritten node is then recursed
	// to catch nested json usage.
	if v.allowed.streaming {
		if r, ok := rewriteStreamingCall(mi, v.jsonPkg); ok {
			return v.GoVisitor.VisitMethodInvocation(r, p)
		}
	}

	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if v.allowed.marshalIndent {
		if r, ok := v.rewriteMarshalIndent(mi); ok {
			return r
		}
	}
	return mi
}

// Rewrites json.NewEncoder(w).Encode(v) to json.MarshalWrite(w, v) and
// json.NewDecoder(r).Decode(&v) to json.UnmarshalRead(r, &v), preserving the
// receiver and leading whitespace.
func rewriteStreamingCall(mi *java.MethodInvocation, jsonPkg string) (*java.MethodInvocation, bool) {
	inner, ok := handledStreamingChain(mi, jsonPkg)
	if !ok {
		return nil, false
	}
	fn, _ := streamingTarget(mi.Name.Name)

	stream := inner.Arguments.Elements[0] // the io.Writer / io.Reader
	value := mi.Arguments.Elements[0]     // the value / pointer being encoded or decoded

	// The writer keeps its position after the paren and the value gets a space
	// after the comma, both carrying their original trailing trivia so a comment
	// is not dropped.
	return &java.MethodInvocation{
		Prefix: mi.Prefix,
		Select: inner.Select,
		Name:   &java.Identifier{Name: fn},
		Arguments: java.Container[java.Expression]{
			Before: inner.Arguments.Before,
			Elements: []java.RightPadded[java.Expression]{
				{Element: stream.Element, After: stream.After},
				{Element: withLeadingSpace(value.Element), After: value.After},
			},
		},
	}, true
}

// Rewrites json.MarshalIndent(v, prefix, indent) to json.Marshal(v,
// jsontext.WithIndent(indent), jsontext.WithIndentPrefix(prefix)).
func (v *mechanicalMigrator) rewriteMarshalIndent(mi *java.MethodInvocation) (*java.MethodInvocation, bool) {
	if selectIdentName(mi) != v.jsonPkg || mi.Name == nil || mi.Name.Name != "MarshalIndent" {
		return nil, false
	}
	if len(mi.Arguments.Elements) != 3 {
		return nil, false
	}
	value := mi.Arguments.Elements[0]
	prefixArg := mi.Arguments.Elements[1].Element
	indentArg := mi.Arguments.Elements[2].Element

	recipegolang.MaybeAddImport(v, "encoding/json/jsontext", nil, false)

	return &java.MethodInvocation{
		Prefix: mi.Prefix,
		Select: mi.Select,
		Name:   &java.Identifier{Name: "Marshal"},
		Arguments: java.Container[java.Expression]{
			Before: mi.Arguments.Before,
			Elements: []java.RightPadded[java.Expression]{
				{Element: value.Element, After: value.After},
				{Element: jsontextOption("WithIndent", indentArg), After: java.EmptySpace},
				{Element: jsontextOption("WithIndentPrefix", prefixArg), After: java.EmptySpace},
			},
		},
	}, true
}

// Builds a jsontext.<fn>(arg) option call, spaced to follow a comma in an
// argument list.
func jsontextOption(fn string, arg java.Expression) *java.MethodInvocation {
	return &java.MethodInvocation{
		Prefix: java.SingleSpace,
		Select: &java.RightPadded[java.Expression]{Element: &java.Identifier{Name: "jsontext"}},
		Name:   &java.Identifier{Name: fn},
		Arguments: java.Container[java.Expression]{
			Elements: []java.RightPadded[java.Expression]{
				{Element: withExprWhitespace(arg, ""), After: java.EmptySpace},
			},
		},
	}
}

// Reports whether cu contains any json touchpoint the allowed set cannot
// migrate or that would leave json unreferenced, so the whole file is left
// unchanged.
func mechanicalFileBlocked(cu *golang.CompilationUnit, jsonPkg string, allowed mechanicalSet) bool {
	scan := visitor.Init(&mechanicalBlockerScan{
		jsonPkg: jsonPkg,
		allowed: allowed,
		handled: map[uuid.UUID]struct{}{},
	})
	scan.Visit(cu, nil)
	// A migratable file must still reference json through a v2 package function
	// after the rename, otherwise the renamed import is left unused.
	return scan.blocked || !scan.jsonReferencedAfter
}

type mechanicalBlockerScan struct {
	visitor.GoVisitor
	jsonPkg             string
	allowed             mechanicalSet
	insideStruct        int
	blocked             bool
	jsonReferencedAfter bool
	handled             map[uuid.UUID]struct{}
}

func (s *mechanicalBlockerScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	if s.blocked {
		return mi
	}
	// Record a fluent chain's constructor so the pre-order recursion does not
	// separately classify it as a standalone json.NewEncoder call.
	if inner, ok := handledStreamingChain(mi, s.jsonPkg); ok {
		s.handled[inner.ID] = struct{}{}
		if !s.allowed.streaming {
			s.blocked = true
			return mi
		}
		s.jsonReferencedAfter = true
		return s.GoVisitor.VisitMethodInvocation(mi, p)
	}
	if _, isHandledCtor := s.handled[mi.ID]; isHandledCtor {
		return s.GoVisitor.VisitMethodInvocation(mi, p)
	}

	kind, isJson := s.classifyCall(mi)
	if isJson {
		if !s.isAllowed(kind) {
			s.blocked = true
			return mi
		}
		if kind == constructMarshalIndent {
			s.jsonReferencedAfter = true
		}
	}
	return s.GoVisitor.VisitMethodInvocation(mi, p)
}

// Categorizes a method invocation as a json touchpoint, returning false when the
// call is unrelated to encoding/json.
func (s *mechanicalBlockerScan) classifyCall(mi *java.MethodInvocation) (constructKind, bool) {
	if mi.Name != nil && selectIdentName(mi) == s.jsonPkg {
		// A MarshalIndent call is migratable only with three plain arguments,
		// since the rewrite reorders them and cannot safely carry argument
		// comments.
		if mi.Name.Name == "MarshalIndent" && len(mi.Arguments.Elements) == 3 && !marshalIndentHasArgComments(mi) {
			return constructMarshalIndent, true
		}
		return constructOther, true
	}
	declaring := declaringFQN(mi)
	if declaring == "encoding/json" || strings.HasPrefix(declaring, "encoding/json.") {
		return constructOther, true
	}
	return constructOther, false
}

func (s *mechanicalBlockerScan) isAllowed(k constructKind) bool {
	switch k {
	case constructStreaming:
		return s.allowed.streaming
	case constructMarshalIndent:
		return s.allowed.marshalIndent
	}
	return false
}

func (s *mechanicalBlockerScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if s.blocked {
		return fa
	}
	// Any json.<Name> reference that is not a call is an exported type or value
	// this recipe cannot rewrite, such as Encoder, Decoder, RawMessage, or
	// Number.
	if ident, ok := fa.Target.(*java.Identifier); ok && ident.Name == s.jsonPkg {
		s.blocked = true
		return fa
	}
	return s.GoVisitor.VisitFieldAccess(fa, p)
}

func (s *mechanicalBlockerScan) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	if s.blocked {
		return md
	}
	if md.Name != nil && (md.Name.Name == "MarshalJSON" || md.Name.Name == "UnmarshalJSON") {
		s.blocked = true
		return md
	}
	return s.GoVisitor.VisitMethodDeclaration(md, p)
}

func (s *mechanicalBlockerScan) VisitStructType(st *golang.StructType, p any) java.J {
	s.insideStruct++
	st = s.GoVisitor.VisitStructType(st, p).(*golang.StructType)
	s.insideStruct--
	return st
}

func (s *mechanicalBlockerScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	if s.blocked {
		return vd
	}
	// omitempty tags only appear on struct fields, and the two field types below
	// change JSON representation in v2 and need review.
	if hasOmitemptyTag(vd) || (s.insideStruct > 0 && isByteArrayOrDurationField(vd)) {
		s.blocked = true
		return vd
	}
	return s.GoVisitor.VisitVariableDeclarations(vd, p)
}

// Reports whether any MarshalIndent argument carries a comment in its own prefix
// or trailing space, which the reordering rewrite cannot preserve.
func marshalIndentHasArgComments(mi *java.MethodInvocation) bool {
	for _, e := range mi.Arguments.Elements {
		if len(e.After.Comments) > 0 {
			return true
		}
		if e.Element != nil && len(e.Element.GetPrefix().Comments) > 0 {
			return true
		}
	}
	return false
}
