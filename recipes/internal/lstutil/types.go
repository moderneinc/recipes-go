/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package lstutil

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

// The types a rewritten node carries when the recipe has to name one itself.
var (
	ErrorType  = &java.JavaTypeClass{Kind: "Interface", FullyQualifiedName: "error"}
	StringType = &java.JavaTypePrimitive{Keyword: "String"}
	BoolType   = &java.JavaTypePrimitive{Keyword: "boolean"}
	VoidType   = &java.JavaTypePrimitive{Keyword: "void"}
)

// NamedType returns the type a fully-qualified Go name carries. A package
// function's declaring type is its import path, so this covers both a named type
// ("strings.Builder") and the qualifier identifier of a `pkg.Fn(...)` call.
func NamedType(fqn string) *java.JavaTypeClass {
	return &java.JavaTypeClass{Kind: "Class", FullyQualifiedName: fqn}
}

// FuncType types a call whose signature the recipe has to state itself. A nil
// returnType means unknown, which is how a rewrite that cannot name one leaves
// it. Prefer MethodOn where the receiver resolves, since that reads the
// signature the parser already recorded.
func FuncType(declaringFQN, name string, returnType java.JavaType) *java.JavaTypeMethod {
	return &java.JavaTypeMethod{DeclaringType: NamedType(declaringFQN), Name: name, ReturnType: returnType}
}

// ReturnTypeOf gives mi's return type, or nil when mi is unattributed.
func ReturnTypeOf(mi *java.MethodInvocation) java.JavaType {
	if mi == nil || mi.MethodType == nil {
		return nil
	}
	return mi.MethodType.ReturnType
}

// MethodOn returns recv's method of that name, off the receiver's own
// attribution. Nil when recv is unresolved or declares no such method.
func MethodOn(recv java.JavaType, name string) *java.JavaTypeMethod {
	class := matcher.AsClass(recv)
	if class == nil {
		return nil
	}
	for _, m := range class.Methods {
		if m != nil && m.Name == name {
			return m
		}
	}
	return nil
}

// FieldOn returns the type of recv's field of that name, or nil when recv is
// unresolved or declares no such field.
func FieldOn(recv java.JavaType, name string) java.JavaType {
	class := matcher.AsClass(recv)
	if class == nil {
		return nil
	}
	for _, m := range class.Members {
		if m != nil && m.Name == name {
			return m.Type
		}
	}
	return nil
}

// GoBasicTypes are Go's predeclared value types, which the parser attributes by
// their Go name rather than as a primitive keyword. Use `matcher.IsSameGoType`
// to compare two of them: it resolves the `byte`/`uint8` and `rune`/`int32`
// spellings, which are attributed distinctly.
var GoBasicTypes = map[string]bool{
	"bool": true, "string": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"byte": true, "rune": true,
	"float32": true, "float64": true, "complex64": true, "complex128": true,
}
