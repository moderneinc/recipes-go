/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// atomicMethodMapping maps deprecated sync/atomic free-function prefixes
// to their type-safe method equivalents.
// e.g. "Add" → "Add", "Load" → "Load", "Store" → "Store",
//
//	"CompareAndSwap" → "CompareAndSwap", "Swap" → "Swap"
var atomicMethodPrefixes = []string{
	"CompareAndSwap", // must be before "Swap" to match first
	"Swap",
	"Add",
	"Load",
	"Store",
}

// atomicTypeSuffixes lists the type suffixes that can follow a method prefix.
var atomicTypeSuffixes = map[string]bool{
	"Int32":   true,
	"Int64":   true,
	"Uint32":  true,
	"Uint64":  true,
	"Uintptr": true,
	"Pointer": true,
}

// parseAtomicFunc splits a function name like "AddInt32" into ("Add", "Int32").
// Returns ("", "") if the name does not match a known atomic function.
func parseAtomicFunc(name string) (method, typeSuffix string) {
	for _, prefix := range atomicMethodPrefixes {
		if strings.HasPrefix(name, prefix) {
			suffix := name[len(prefix):]
			if atomicTypeSuffixes[suffix] {
				return prefix, suffix
			}
		}
	}
	return "", ""
}

// UseAtomicTypes transforms deprecated `sync/atomic` free-function calls
// such as `atomic.AddInt32(&x, 1)` into method calls on the type-safe
// atomic types introduced in Go 1.19, e.g. `x.Add(1)`.
type UseAtomicTypes struct {
	recipe.Base
}

func (r *UseAtomicTypes) Name() string {
	return "org.openrewrite.golang.codequality.UseAtomicTypes"
}
func (r *UseAtomicTypes) DisplayName() string { return "Use atomic types" }
func (r *UseAtomicTypes) Description() string {
	return "Transform deprecated `sync/atomic` free-function calls into method calls on the type-safe atomic types introduced in Go 1.19 (e.g. `atomic.Int32`)."
}
func (r *UseAtomicTypes) Tags() []string { return []string{"style", "concurrency"} }

func (r *UseAtomicTypes) Editor() recipe.TreeVisitor {
	return visitor.Init(&useAtomicTypesVisitor{})
}

type useAtomicTypesVisitor struct {
	visitor.GoVisitor
}

func (v *useAtomicTypesVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "atomic" {
		return mi
	}

	method, _ := parseAtomicFunc(mi.Name.Name)
	if method == "" {
		return mi
	}

	args := mi.Arguments.Elements
	if len(args) == 0 {
		return mi
	}

	// First argument must be &x (AddressOf unary); strip the & to get the receiver.
	firstArg := args[0].Element
	addrOf, ok := firstArg.(*java.Unary)
	if !ok || addrOf.Operator.Element != java.AddressOf {
		// Cannot transform if first arg is not &x — add markup instead.
		mi = mi.WithMarkers(java.MarkupWarn(mi.Markers, "deprecated sync/atomic function; use type-safe atomic types (Go 1.19+)"))
		return mi
	}

	receiver := addrOf.Operand

	// Build new method invocation: receiver.Method(remaining args...)
	newName := &java.Identifier{
		ID:   mi.Name.ID,
		Name: method,
	}

	// Remaining args (everything after the first &x)
	var newArgs []java.RightPadded[java.Expression]
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if i == 1 {
			// Remove leading space that was after the comma separator
			arg.Element = setAtomicExprPrefix(arg.Element, java.EmptySpace)
		}
		newArgs = append(newArgs, arg)
	}

	newMI := &java.MethodInvocation{
		ID:     mi.ID,
		Prefix: mi.Prefix,
		Select: &java.RightPadded[java.Expression]{
			Element: setAtomicExprPrefix(receiver, ident.Prefix),
		},
		Name: newName,
		Arguments: java.Container[java.Expression]{
			Before:   mi.Arguments.Before,
			Elements: newArgs,
			Markers:  mi.Arguments.Markers,
		},
		MethodType: mi.MethodType,
	}

	return newMI
}

// setAtomicExprPrefix sets the prefix on common expression node types.
func setAtomicExprPrefix(expr java.Expression, prefix java.Space) java.Expression {
	switch e := expr.(type) {
	case *java.Identifier:
		return e.WithPrefix(prefix)
	case *java.Literal:
		return e.WithPrefix(prefix)
	case *java.FieldAccess:
		return e.WithPrefix(prefix)
	case *java.Unary:
		return e.WithPrefix(prefix)
	default:
		return expr
	}
}
