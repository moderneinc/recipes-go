/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"reflect"
	"strings"

	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Queues the encoding/json to encoding/json/v2 import rename on the current
// file, the shared import step every construct rewrite defers to. Queued before
// recursion so the rename runs first and never touches a jsontext import added
// during it.
func queueImportSwapToV2(v visitor.AfterVisitsProvider) {
	v.DoAfterVisit((&recipegolang.RenamePackage{
		OldPackagePath: "encoding/json",
		NewPackagePath: "encoding/json/v2",
	}).Editor())
}

// Reports whether cu imports any encoding/json/... sub-path, such as
// encoding/json/v2 or encoding/json/jsontext.
func importsEncodingJsonSubpath(cu *golang.CompilationUnit) bool {
	if cu.Imports == nil {
		return false
	}
	for _, imp := range cu.Imports.Elements {
		if path, ok := importPath(imp.Element); ok && strings.HasPrefix(path, "encoding/json/") {
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

// Maps a streaming method to its v2 package function and the constructor that
// must front it.
func streamingTarget(method string) (fn, ctor string) {
	switch method {
	case "Encode":
		return "MarshalWrite", "NewEncoder"
	case "Decode":
		return "UnmarshalRead", "NewDecoder"
	}
	return "", ""
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
