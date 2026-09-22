/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package expstd

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Reports x/exp usage alongside the standard library form that replaces it.
type FindXExpUsage struct {
	recipe.Base
}

func (r *FindXExpUsage) Name() string {
	return "org.openrewrite.golang.migration.FindXExpUsage"
}
func (r *FindXExpUsage) DisplayName() string {
	return "Find `golang.org/x/exp` usage the standard library covers"
}
func (r *FindXExpUsage) Description() string {
	return "Mark every `golang.org/x/exp/slices`, `/maps` and `/constraints` call and type reference with its standard-library replacement, and warn on the two shapes that do not have one: the numeric constraints, and a boolean comparator passed to a sort function."
}
func (r *FindXExpUsage) Tags() []string {
	return []string{"search", "migration", "stdlib", "slices", "maps"}
}

func (r *FindXExpUsage) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "exptostd", Tool: diagnostic.GolangciLint, HasFix: false},
	}
}

func (r *FindXExpUsage) Editor() recipe.TreeVisitor {
	return visitor.Init(&findXExpUsageVisitor{})
}

type findXExpUsageVisitor struct {
	visitor.GoVisitor
	slicesPkg      string
	mapsPkg        string
	constraintsPkg string
}

func (v *findXExpUsageVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.slicesPkg = pathswap.Qualifier(cu, expSlices)
	v.mapsPkg = pathswap.Qualifier(cu, expMaps)
	v.constraintsPkg = pathswap.Qualifier(cu, expConstraints)
	if v.slicesPkg == "" && v.mapsPkg == "" && v.constraintsPkg == "" {
		return cu
	}
	return v.GoVisitor.VisitCompilationUnit(cu, p)
}

func (v *findXExpUsageVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)

	if name, ok := qualifiedCall(mi, v.slicesPkg); ok {
		if lessFuncTakers[name] {
			for _, arg := range realArgs(mi) {
				if returnsBool(arg) {
					return mi.WithMarkers(java.MarkupWarn(mi.Markers,
						"slices."+name+" takes a three-way `cmp func(a, b E) int`; this boolean comparator is the pre-2023 x/exp signature and has to be converted by hand"))
				}
			}
		}
		return mi.WithMarkers(java.MarkupInfo(mi.Markers, "replaceable with slices."+name+" from the standard library (Go 1.21)"))
	}

	if name, ok := qualifiedCall(mi, v.mapsPkg); ok {
		return mi.WithMarkers(java.MarkupInfo(mi.Markers, mapsReplacement(name)))
	}
	return mi
}

func (v *findXExpUsageVisitor) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)
	name, ok := qualifiedRef(fa, v.constraintsPkg)
	if !ok {
		return fa
	}
	if expOnlyConstraints[name] {
		return fa.WithMarkers(java.MarkupWarn(fa.Markers,
			"constraints."+name+" has no standard-library counterpart; declare the type set in your own package to drop golang.org/x/exp"))
	}
	if name == "Ordered" {
		return fa.WithMarkers(java.MarkupInfo(fa.Markers, "replaceable with cmp.Ordered from the standard library (Go 1.21)"))
	}
	return fa
}

// mapsReplacement names the standard-library form of an x/exp/maps function.
func mapsReplacement(name string) string {
	switch name {
	case "Keys", "Values":
		return "maps." + name + " returns an iter.Seq in the standard library; wrap it as slices.Collect(maps." + name + "(m)) (Go 1.23)"
	case "Clear":
		return "replaceable with the clear(m) builtin (Go 1.21)"
	}
	return "replaceable with maps." + name + " from the standard library (Go 1.21)"
}
