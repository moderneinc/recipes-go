/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/moderneinc/recipes-go/recipes-code-quality/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveRedundantTypeConversion is a search-only recipe that finds
// type conversions like `int(x)` where x is already the target type.
// Without full type attribution, this recipe flags all single-arg
// type conversions where the function name matches a builtin type
// and the argument is a variable of the same name pattern.
// Staticcheck: S1021
type RemoveRedundantTypeConversion struct {
	recipe.Base
}

func (r *RemoveRedundantTypeConversion) Name() string {
	return "org.openrewrite.golang.codequality.RemoveRedundantTypeConversion"
}
func (r *RemoveRedundantTypeConversion) DisplayName() string {
	return "Find potentially redundant type conversion"
}
func (r *RemoveRedundantTypeConversion) Description() string {
	return "Find type conversions like `int(x)` that may be redundant if x is already the target type. Requires type attribution for full accuracy."
}
func (r *RemoveRedundantTypeConversion) Tags() []string { return []string{"cleanup"} }

func (r *RemoveRedundantTypeConversion) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "S1021", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *RemoveRedundantTypeConversion) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeRedundantTypeConversionVisitor{})
}

type removeRedundantTypeConversionVisitor struct {
	visitor.GoVisitor
}

func (v *removeRedundantTypeConversionVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	// Type conversions in Go look like function calls: int(x), string(b), etc.
	// They have no Select (no receiver) and the Name is a builtin type.
	if mi.Select != nil {
		return mi
	}
	if !isBuiltinType(mi.Name.Name) {
		return mi
	}

	// Must have exactly one argument
	var argCount int
	for _, a := range mi.Arguments.Elements {
		if _, isEmpty := a.Element.(*java.Empty); !isEmpty {
			argCount++
		}
	}
	if argCount != 1 {
		return mi
	}

	// Mark as a potential redundant conversion (search only).
	// Full accuracy requires type attribution to confirm the arg type matches.
	mi = mi.WithMarkers(java.MarkupInfo(mi.Markers, "potentially redundant type conversion"))
	return mi
}

var builtinTypes = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
	"string": true, "byte": true, "rune": true,
	"bool": true, "uintptr": true,
}

func isBuiltinType(name string) bool {
	return builtinTypes[name]
}
