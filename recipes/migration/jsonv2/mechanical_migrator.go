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
// migrated only when this recipe rewrites at least one construct and every other
// json touchpoint still compiles under v2.
type mechanicalSet struct {
	streaming     bool // json.NewEncoder(w).Encode(v) / json.NewDecoder(r).Decode(&v)
	marshalIndent bool // json.MarshalIndent(v, prefix, indent)
}

// Classification of a single json touchpoint.
type constructKind int

const (
	constructNone          constructKind = iota // not an encoding/json touchpoint
	constructSurvivor                           // Marshal/Unmarshal/Marshaler/Unmarshaler, unchanged in v2
	constructStreaming                          // json.NewEncoder(w).Encode(v) chain
	constructMarshalIndent                      // json.MarshalIndent(...)
	constructRemoved                            // a v1 symbol v2 dropped, so the import cannot be swapped
)

// Applies the allowed mechanical rewrites and swaps the import, per file, only
// when the file is fully migratable by its allowed set.
type mechanicalMigrator struct {
	visitor.GoVisitor
	allowed mechanicalSet
	jsonPkg string
}

func (v *mechanicalMigrator) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	jsonPkg := regularJsonPackage(cu)
	if jsonPkg == "" {
		return cu
	}

	// A file already importing encoding/json/v2 is skipped so the swap does not
	// produce a duplicate import.
	if importsEncodingJsonV2(cu) {
		return cu
	}

	// Leave the file untouched unless this recipe rewrites a construct here and
	// the import can be swapped without stranding a v1 symbol v2 removed.
	if mechanicalFileBlocked(cu, jsonPkg, v.allowed) {
		return cu
	}

	v.jsonPkg = jsonPkg
	queueImportSwapToV2(v)

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

// Reports whether cu should be left unchanged, either because this recipe
// rewrites nothing in it or because a v1 symbol v2 removed would be stranded by
// the import swap.
func mechanicalFileBlocked(cu *golang.CompilationUnit, jsonPkg string, allowed mechanicalSet) bool {
	scan := visitor.Init(&mechanicalBlockerScan{
		jsonPkg: jsonPkg,
		allowed: allowed,
		handled: map[uuid.UUID]struct{}{},
	})
	scan.Visit(cu, nil)
	// A migratable file must have at least one construct this recipe rewrites,
	// otherwise the import swap would change nothing but the import.
	return scan.blocked || !scan.jsonReferencedAfter
}

type mechanicalBlockerScan struct {
	visitor.GoVisitor
	jsonPkg             string
	allowed             mechanicalSet
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
		if isV2SurvivingJsonName(mi.Name.Name) {
			return constructSurvivor, true
		}
		// Compact, HTMLEscape, Indent, Valid, a stored NewEncoder/NewDecoder, and
		// every other package function v2 dropped.
		return constructRemoved, true
	}
	// A method call on a v1 Encoder/Decoder value, which v2 relocated to jsontext.
	declaring := declaringFQN(mi)
	if declaring == "encoding/json" || strings.HasPrefix(declaring, "encoding/json.") {
		return constructRemoved, true
	}
	return constructNone, false
}

func (s *mechanicalBlockerScan) isAllowed(k constructKind) bool {
	switch k {
	case constructSurvivor:
		return true
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
	// A json.<Name> reference to a type or value v2 removed cannot survive the
	// import swap; RawMessage, Number, Encoder, Decoder, Delim, Token, and the
	// error types were all removed or relocated to jsontext.
	if jsonFieldAccessBlocks(fa, s.jsonPkg) {
		s.blocked = true
		return fa
	}
	return s.GoVisitor.VisitFieldAccess(fa, p)
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
