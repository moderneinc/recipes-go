/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package naming

import (
	"github.com/openrewrite/rewrite/pkg/recipe"
	"github.com/openrewrite/rewrite/pkg/tree"
	"github.com/openrewrite/rewrite/pkg/visitor"
)

// FindBadPackageName finds packages named "util", "utils", "common", "shared",
// "misc", or "helpers". These generic names are anti-patterns in Go because
// they do not convey meaning and encourage dumping unrelated code together.
type FindBadPackageName struct {
	recipe.Base
}

func (r *FindBadPackageName) Name() string {
	return "org.openrewrite.golang.codequality.FindBadPackageName"
}
func (r *FindBadPackageName) DisplayName() string { return "Find bad package names" }
func (r *FindBadPackageName) Description() string {
	return "Find packages named util, utils, common, shared, misc, or helpers which are anti-patterns in Go."
}
func (r *FindBadPackageName) Tags() []string { return []string{"naming"} }

func (r *FindBadPackageName) Editor() recipe.TreeVisitor {
	return visitor.Init(&findBadPackageNameVisitor{})
}

var badPackageNames = map[string]bool{
	"util":    true,
	"utils":   true,
	"common":  true,
	"shared":  true,
	"misc":    true,
	"helpers": true,
}

type findBadPackageNameVisitor struct {
	visitor.GoVisitor
}

func (v *findBadPackageNameVisitor) VisitCompilationUnit(cu *tree.CompilationUnit, p any) tree.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*tree.CompilationUnit)

	if cu.PackageDecl == nil {
		return cu
	}

	pkgName := cu.PackageDecl.Element.Name
	if !badPackageNames[pkgName] {
		return cu
	}

	pkg := *cu.PackageDecl
	pkg.Element = pkg.Element.WithMarkers(
		tree.FoundSearchResult(pkg.Element.Markers, "package name is too generic; consider a more descriptive name"),
	)
	cu = cu.WithPackageDecl(&pkg)
	return cu
}
