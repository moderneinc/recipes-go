/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"reflect"
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Queues the encoding/json to encoding/json/v2 import swap on the current file,
// the shared import step every construct rewrite defers to. It rewrites only the
// exact encoding/json import, so an existing encoding/json/jsontext sub-path
// import is left intact and its alias is preserved.
func queueImportSwapToV2(v visitor.AfterVisitsProvider) {
	v.DoAfterVisit(visitor.Init(&importSwapToV2Visitor{}))
}

// Applies to cu any import operations the visit queued via DoAfterVisit so they
// land as part of the returned tree, which the Moderne CLI's RPC applies where it
// does not run the recipe runner's separate DoAfterVisit drain.
func drainQueuedImports(v visitor.AfterVisitsProvider, cu *golang.CompilationUnit, p any) *golang.CompilationUnit {
	if drained, ok := visitor.DrainAfterVisits(v, cu, p).(*golang.CompilationUnit); ok {
		return drained
	}
	return cu
}

type importSwapToV2Visitor struct {
	visitor.GoVisitor
}

func (v *importSwapToV2Visitor) VisitImport(imp *java.Import, p any) java.J {
	imp = v.GoVisitor.VisitImport(imp, p).(*java.Import)
	lit, ok := imp.Qualid.(*java.Literal)
	if !ok {
		return imp
	}
	if path, ok := importPath(imp); !ok || path != "encoding/json" {
		return imp
	}
	newLit := *lit
	newLit.Value = "encoding/json/v2"
	newLit.Source = strings.Replace(lit.Source, "encoding/json", "encoding/json/v2", 1)
	c := *imp
	c.Qualid = &newLit
	return &c
}

// Reports whether cu already imports encoding/json/v2, so the swap would produce
// a duplicate import and the file is left unchanged.
func importsEncodingJsonV2(cu *golang.CompilationUnit) bool {
	if cu.Imports == nil {
		return false
	}
	for _, imp := range cu.Imports.Elements {
		if path, ok := importPath(imp.Element); ok && path == "encoding/json/v2" {
			return true
		}
	}
	return false
}

// Reports whether mi is a json.NewEncoder(w).Encode(v) or
// json.NewDecoder(r).Decode(&v) chain on the given local json package name,
// returning the inner constructor call when it is.
func handledStreamingChain(mi *java.MethodInvocation, jsonPkg string) (*java.MethodInvocation, bool) {
	if mi.Name == nil {
		return nil, false
	}
	_, ctor := streamingTarget(mi.Name.Name)
	if ctor == "" || mi.Select == nil {
		return nil, false
	}
	inner, ok := mi.Select.Element.(*java.MethodInvocation)
	if !ok || inner.Name == nil || inner.Name.Name != ctor || inner.Select == nil {
		return nil, false
	}
	recv, ok := inner.Select.Element.(*java.Identifier)
	if !ok || recv.Name != jsonPkg {
		return nil, false
	}
	if len(inner.Arguments.Elements) != 1 || len(mi.Arguments.Elements) != 1 {
		return nil, false
	}
	return inner, true
}

// Maps a streaming method to its v2 package function and the jsontext
// constructor that fronts it; the jsontext codec preserves v1's streaming
// contract (the trailing newline on encode, a single value on decode).
func streamingTarget(method string) (fn, ctor string) {
	switch method {
	case "Encode":
		return "MarshalEncode", "NewEncoder"
	case "Decode":
		return "UnmarshalDecode", "NewDecoder"
	}
	return "", ""
}

// Reports whether cu declares a struct field whose bare-v2 encoding differs from
// v1: a time.Duration, which v2 marshals as a runtime error without an explicit
// format, or a fixed [N]byte array, which v2 encodes as a base64 string rather
// than a number array. The default path leaves such a file for review, while the
// compat path migrates it, since DefaultOptionsV1 restores the v1 encoding.
func fileNeedsV1Compat(cu *golang.CompilationUnit) bool {
	scan := visitor.Init(&v1CompatFieldScan{})
	scan.Visit(cu, nil)
	return scan.found
}

type v1CompatFieldScan struct {
	visitor.GoVisitor
	insideStruct int
	found        bool
}

func (s *v1CompatFieldScan) VisitStructType(st *golang.StructType, p any) java.J {
	s.insideStruct++
	st = s.GoVisitor.VisitStructType(st, p).(*golang.StructType)
	s.insideStruct--
	return st
}

func (s *v1CompatFieldScan) VisitVariableDeclarations(vd *java.VariableDeclarations, p any) java.J {
	if s.insideStruct > 0 && (isDurationField(vd) || isFixedByteArrayField(vd)) {
		s.found = true
	}
	return s.GoVisitor.VisitVariableDeclarations(vd, p)
}

// Reports whether a struct field is a fixed-size [N]byte array, whose base64
// encoding under v2 differs from v1's number-array encoding. A []byte slice is
// excluded, since it encodes as base64 under both v1 and v2.
func isFixedByteArrayField(vd *java.VariableDeclarations) bool {
	at, ok := vd.TypeExpr.(*golang.ArrayType)
	if !ok || at.Length.Element == nil {
		return false
	}
	elem, ok := at.ElementType.(*java.Identifier)
	return ok && (elem.Name == "byte" || elem.Name == "uint8")
}

// Reports whether a struct field is a time.Duration, resolving an aliased time
// import through the type system.
func isDurationField(vd *java.VariableDeclarations) bool {
	if fa, ok := vd.TypeExpr.(*java.FieldAccess); ok {
		if pkg, ok := fa.Target.(*java.Identifier); ok && pkg.Name == "time" &&
			fa.Name.Element != nil && fa.Name.Element.Name == "Duration" {
			return true
		}
	}
	for _, decl := range vd.Variables {
		if d := decl.Element; d != nil && d.Name != nil {
			if fq, ok := d.Name.Type.(java.FullyQualified); ok && fq.GetFullyQualifiedName() == "time.Duration" {
				return true
			}
		}
	}
	return false
}

// Reports whether a json.<name> identifier still exists in encoding/json/v2, so
// that a reference to it compiles unchanged after the import swap; every other
// v1 export was removed or relocated to jsontext.
func isV2SurvivingJsonName(name string) bool {
	switch name {
	case "Marshal", "Unmarshal", "Marshaler", "Unmarshaler":
		return true
	}
	return false
}

// Returns the local name a file uses for a regular or aliased encoding/json
// import, or an empty string when the package is absent or imported blank or
// dot, none of which is migratable.
func regularJsonPackage(cu *golang.CompilationUnit) string {
	jsonPkg := localJsonPackage(cu)
	if jsonPkg == "" || jsonPkg == "_" || jsonPkg == "." {
		return ""
	}
	return jsonPkg
}

// Reports whether fa is a json.<Name> reference to a symbol v2 removed, which
// cannot survive the import swap and so blocks the file.
func jsonFieldAccessBlocks(fa *java.FieldAccess, jsonPkg string) bool {
	ident, ok := fa.Target.(*java.Identifier)
	if !ok || ident.Name != jsonPkg {
		return false
	}
	return fa.Name.Element == nil || !isV2SurvivingJsonName(fa.Name.Element.Name)
}

// Returns expr with a single leading space, preserving any leading comments.
func withLeadingSpace(expr java.Expression) java.Expression {
	return withExprWhitespace(expr, " ")
}

// Returns expr with its leading whitespace set to ws, preserving any leading
// comments, using the node's WithPrefix method reflectively so every expression
// type is handled.
func withExprWhitespace(expr java.Expression, ws string) java.Expression {
	prefix := java.Space{Comments: expr.GetPrefix().Comments, Whitespace: ws}
	m := reflect.ValueOf(expr).MethodByName("WithPrefix")
	if !m.IsValid() {
		return expr
	}
	out := m.Call([]reflect.Value{reflect.ValueOf(prefix)})
	if len(out) == 1 {
		if e, ok := out[0].Interface().(java.Expression); ok {
			return e
		}
	}
	return expr
}
