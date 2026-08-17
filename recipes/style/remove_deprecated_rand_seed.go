/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
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

func (r *RemoveDeprecatedRandSeed) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "SA1019", Tool: diagnostic.Staticcheck, HasFix: true},
	}
}

func (r *RemoveDeprecatedRandSeed) Editor() recipe.TreeVisitor {
	return visitor.Init(&removeDeprecatedRandSeedVisitor{})
}

type removeDeprecatedRandSeedVisitor struct {
	visitor.GoVisitor
	changed bool
}

func (v *removeDeprecatedRandSeedVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)
	if v.changed {
		v.DoAfterVisit((&recipegolang.RemoveUnusedImports{}).Editor())
	}
	return cu
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
	v.changed = true
	return &java.Empty{}
}
