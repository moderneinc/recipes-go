/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// mathRandMethods lists the commonly used math/rand functions that may
// indicate insecure randomness.
var mathRandMethods = map[string]bool{
	"Intn":    true,
	"Int":     true,
	"Int31":   true,
	"Int31n":  true,
	"Int63":   true,
	"Int63n":  true,
	"Float32": true,
	"Float64": true,
	"Seed":    true,
	"Read":    true,
}

// FindMathRand finds usage of `math/rand` functions such as `rand.Intn`,
// `rand.Int`, `rand.Float64`, etc. For security-sensitive code, `crypto/rand`
// should be used instead.
type FindMathRand struct {
	recipe.Base
}

func (r *FindMathRand) Name() string {
	return "org.openrewrite.golang.codequality.FindMathRand"
}
func (r *FindMathRand) DisplayName() string { return "Find math/rand usage" }
func (r *FindMathRand) Description() string {
	return "Find usage of `math/rand` functions. Consider using `crypto/rand` for security-sensitive randomness."
}
func (r *FindMathRand) Tags() []string { return []string{"style", "security"} }

func (r *FindMathRand) Editor() recipe.TreeVisitor {
	return visitor.Init(&findMathRandVisitor{})
}

type findMathRandVisitor struct {
	visitor.GoVisitor
}

func (v *findMathRandVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*tree.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "rand" {
		return mi
	}

	if !mathRandMethods[mi.Name.Name] {
		return mi
	}

	mi = mi.WithMarkers(tree.FoundSearchResult(mi.Markers, "consider using crypto/rand for security-sensitive randomness"))
	return mi
}
