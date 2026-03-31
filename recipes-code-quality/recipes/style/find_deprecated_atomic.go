/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// deprecatedAtomicFuncs lists the sync/atomic free functions that are
// deprecated in favor of the type-safe atomic types introduced in Go 1.19
// (e.g. atomic.Int32, atomic.Int64).
var deprecatedAtomicFuncs = map[string]bool{
	"AddInt32":          true,
	"AddInt64":          true,
	"AddUint32":         true,
	"AddUint64":         true,
	"AddUintptr":        true,
	"CompareAndSwapInt32":  true,
	"CompareAndSwapInt64":  true,
	"CompareAndSwapUint32": true,
	"CompareAndSwapUint64": true,
	"CompareAndSwapUintptr": true,
	"CompareAndSwapPointer": true,
	"LoadInt32":          true,
	"LoadInt64":          true,
	"LoadUint32":         true,
	"LoadUint64":         true,
	"LoadUintptr":        true,
	"LoadPointer":        true,
	"StoreInt32":         true,
	"StoreInt64":         true,
	"StoreUint32":        true,
	"StoreUint64":        true,
	"StoreUintptr":       true,
	"StorePointer":       true,
	"SwapInt32":          true,
	"SwapInt64":          true,
	"SwapUint32":         true,
	"SwapUint64":         true,
	"SwapUintptr":        true,
	"SwapPointer":        true,
}

// FindDeprecatedAtomic finds usage of deprecated `sync/atomic` free functions
// such as `atomic.AddInt32`, `atomic.LoadInt64`, etc. Since Go 1.19, the
// type-safe atomic types (e.g. `atomic.Int32`) should be preferred.
type FindDeprecatedAtomic struct {
	recipe.Base
}

func (r *FindDeprecatedAtomic) Name() string {
	return "org.openrewrite.golang.codequality.FindDeprecatedAtomic"
}
func (r *FindDeprecatedAtomic) DisplayName() string { return "Find deprecated sync/atomic functions" }
func (r *FindDeprecatedAtomic) Description() string {
	return "Find usage of deprecated `sync/atomic` free functions such as `atomic.AddInt32`. Prefer the type-safe atomic types introduced in Go 1.19 (e.g. `atomic.Int32`)."
}
func (r *FindDeprecatedAtomic) Tags() []string { return []string{"style", "concurrency"} }

func (r *FindDeprecatedAtomic) Editor() recipe.TreeVisitor {
	return visitor.Init(&findDeprecatedAtomicVisitor{})
}

type findDeprecatedAtomicVisitor struct {
	visitor.GoVisitor
}

func (v *findDeprecatedAtomicVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "atomic" {
		return mi
	}

	if !deprecatedAtomicFuncs[mi.Name.Name] {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "consider using type-safe atomic types (Go 1.19+)"))
	return mi
}
