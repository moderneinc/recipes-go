/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseDescriptivePackageName finds packages named "util", "utils", "common", "shared",
// "misc", or "helpers". These generic names are anti-patterns in Go because
// they do not convey meaning and encourage dumping unrelated code together.
type UseDescriptivePackageName struct {
	recipe.Base
}

func (r *UseDescriptivePackageName) Name() string {
	return "org.openrewrite.golang.codequality.UseDescriptivePackageName"
}
func (r *UseDescriptivePackageName) DisplayName() string { return "Use descriptive package name" }
func (r *UseDescriptivePackageName) Description() string {
	return "Find packages named util, utils, common, shared, misc, or helpers which are anti-patterns in Go."
}
func (r *UseDescriptivePackageName) Tags() []string { return []string{"naming"} }

func (r *UseDescriptivePackageName) Editor() recipe.TreeVisitor {
	return visitor.Init(&useDescriptivePackageNameVisitor{})
}

// badPackageNames is the set of generic package names that are anti-patterns in Go.
var badPackageNames = map[string]bool{
	"util":    true,
	"utils":   true,
	"common":  true,
	"shared":  true,
	"misc":    true,
	"helpers": true,
}

type useDescriptivePackageNameVisitor struct {
	visitor.GoVisitor
}

func (v *useDescriptivePackageNameVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)

	if cu.PackageDecl == nil {
		return cu
	}

	pkgName := cu.PackageDecl.Element.Name
	if !badPackageNames[pkgName] {
		return cu
	}

	pkg := *cu.PackageDecl
	pkg.Element = pkg.Element.WithMarkers(
		java.MarkupWarn(pkg.Element.Markers, "package name is too generic; consider a more descriptive name"),
	)
	cu = cu.WithPackageDecl(&pkg)
	return cu
}
