/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// atomicMethodPrefixes lists the deprecated sync/atomic free-function prefixes.
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

// FindDeprecatedAtomicFunctions flags deprecated `sync/atomic` free-function calls such as
// `atomic.AddInt32(&x, 1)`. The type-safe atomic types introduced in Go 1.19
// (e.g. `atomic.Int32`) are preferred, but migrating requires retyping the
// variable declaration and rewriting every usage (a data-flow change), so this
// recipe only marks the call sites for review rather than rewriting them.
type FindDeprecatedAtomicFunctions struct {
	recipe.Base
}

func (r *FindDeprecatedAtomicFunctions) Name() string {
	return "org.openrewrite.golang.codequality.FindDeprecatedAtomicFunctions"
}
func (r *FindDeprecatedAtomicFunctions) DisplayName() string {
	return "Find deprecated `sync/atomic` functions"
}
func (r *FindDeprecatedAtomicFunctions) Description() string {
	return "Find deprecated `sync/atomic` free-function calls (e.g. `atomic.AddInt32`) that should be migrated to the type-safe atomic types introduced in Go 1.19 (e.g. `atomic.Int32`)."
}
func (r *FindDeprecatedAtomicFunctions) Tags() []string { return []string{"style", "concurrency"} }

func (r *FindDeprecatedAtomicFunctions) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *FindDeprecatedAtomicFunctions) Editor() recipe.TreeVisitor {
	return visitor.Init(&findDeprecatedAtomicFunctionsVisitor{})
}

type findDeprecatedAtomicFunctionsVisitor struct {
	visitor.GoVisitor
}

func (v *findDeprecatedAtomicFunctionsVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "atomic" {
		return mi
	}

	if method, _ := parseAtomicFunc(mi.Name.Name); method == "" {
		return mi
	}

	return mi.WithMarkers(java.MarkupWarn(mi.Markers, "deprecated sync/atomic function; prefer the type-safe atomic types introduced in Go 1.19 (e.g. atomic.Int32)"))
}
