/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package jsonv2

import (
	"reflect"
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

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

// Reports whether a struct field is a fixed-size [N]byte/[N]uint8 or a
// time.Duration, both of which change JSON representation in v2; a named type
// whose underlying type is one of these is not detected.
func isByteArrayOrDurationField(vd *java.VariableDeclarations) bool {
	if at, ok := vd.TypeExpr.(*golang.ArrayType); ok && at.Length.Element != nil {
		if elem, ok := at.ElementType.(*java.Identifier); ok && (elem.Name == "byte" || elem.Name == "uint8") {
			return true
		}
	}
	if fa, ok := vd.TypeExpr.(*java.FieldAccess); ok {
		if pkg, ok := fa.Target.(*java.Identifier); ok && pkg.Name == "time" && fa.Name.Element.Name == "Duration" {
			return true
		}
	}
	// Resolve through the type system so a time.Duration reached via an aliased
	// time import is caught too.
	for _, decl := range vd.Variables {
		if d := decl.Element; d != nil && d.Name != nil {
			if fq, ok := d.Name.Type.(java.FullyQualified); ok && fq.GetFullyQualifiedName() == "time.Duration" {
				return true
			}
		}
	}
	return false
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
