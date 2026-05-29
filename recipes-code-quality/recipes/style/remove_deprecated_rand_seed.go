/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// RemoveDeprecatedRandSeed removes calls to `rand.Seed()`. As of Go 1.20,
// the global random number generator is automatically seeded, making explicit
// calls to `rand.Seed` unnecessary and deprecated.
type RemoveDeprecatedRandSeed struct {
	recipe.Base
}

func (r *RemoveDeprecatedRandSeed) Name() string {
	return "org.openrewrite.golang.codequality.RemoveDeprecatedRandSeed"
}
func (r *RemoveDeprecatedRandSeed) DisplayName() string { return "Remove deprecated rand.Seed" }
func (r *RemoveDeprecatedRandSeed) Description() string {
	return "Remove calls to `rand.Seed()`. Deprecated since Go 1.20; automatic seeding is used."
}
func (r *RemoveDeprecatedRandSeed) Tags() []string { return []string{"style", "deprecation"} }

func (r *RemoveDeprecatedRandSeed) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeDeprecatedRandSeedVisitor{})
}

type removeDeprecatedRandSeedVisitor struct {
	visitor.GoVisitor
}

func (v *removeDeprecatedRandSeedVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if mi.Select == nil {
		return mi
	}

	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "rand" {
		return mi
	}

	if mi.Name.Name != "Seed" {
		return mi
	}

	// Remove the deprecated call.
	return &java.Empty{}
}
