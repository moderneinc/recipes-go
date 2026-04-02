/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
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

// UseCryptoRand finds usage of `math/rand` functions such as `rand.Intn`,
// `rand.Int`, `rand.Float64`, etc. For security-sensitive code, `crypto/rand`
// should be used instead.
type UseCryptoRand struct {
	recipe.Base
}

func (r *UseCryptoRand) Name() string {
	return "org.openrewrite.golang.codequality.UseCryptoRand"
}
func (r *UseCryptoRand) DisplayName() string { return "Use crypto/rand" }
func (r *UseCryptoRand) Description() string {
	return "Find usage of `math/rand` functions. Consider using `crypto/rand` for security-sensitive randomness."
}
func (r *UseCryptoRand) Tags() []string { return []string{"style", "security"} }

func (r *UseCryptoRand) Editor() recipe.TreeVisitor {
	return visitor.Init(&useCryptoRandVisitor{})
}

type useCryptoRandVisitor struct {
	visitor.GoVisitor
}

func (v *useCryptoRandVisitor) VisitMethodInvocation(mi *tree.MethodInvocation, p any) tree.J {
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

	mi = mi.WithMarkers(tree.MarkupInfo(mi.Markers, "consider using crypto/rand for security-sensitive randomness"))
	return mi
}
