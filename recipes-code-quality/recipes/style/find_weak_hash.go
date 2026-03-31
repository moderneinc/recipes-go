/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// weakHashPackages lists the packages that provide weak hash functions.
var weakHashPackages = map[string]bool{
	"md5":  true,
	"sha1": true,
}

// weakHashMethods lists the methods on those packages to flag.
var weakHashMethods = map[string]bool{
	"New": true,
	"Sum": true,
}

// FindWeakHash finds usage of weak hash functions such as `md5.New()`,
// `md5.Sum()`, `sha1.New()`, and `sha1.Sum()`. SHA-256 or stronger
// should be used instead.
type FindWeakHash struct {
	recipe.Base
}

func (r *FindWeakHash) Name() string {
	return "org.openrewrite.golang.codequality.FindWeakHash"
}
func (r *FindWeakHash) DisplayName() string { return "Find weak hash functions" }
func (r *FindWeakHash) Description() string {
	return "Find usage of weak hash functions (md5, sha1). Use SHA-256 or stronger instead."
}
func (r *FindWeakHash) Tags() []string { return []string{"style", "security"} }

func (r *FindWeakHash) Editor() recipe.TreeVisitor {
	return visitor.Init(&findWeakHashVisitor{})
}

type findWeakHashVisitor struct {
	visitor.GoVisitor
}

func (v *findWeakHashVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || !weakHashPackages[ident.Name] {
		return mi
	}

	if !weakHashMethods[mi.Name.Name] {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "weak hash function; use SHA-256 or stronger"))
	return mi
}
