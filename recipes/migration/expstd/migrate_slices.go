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

var slicesRules = []pathswap.Rule{{Old: expSlices, New: stdSlices}}

// Repoints golang.org/x/exp/slices at the standard library.
type MigrateXExpSlicesToStdlib struct {
	recipe.ScanningBase
}

func (r *MigrateXExpSlicesToStdlib) Name() string {
	return "org.openrewrite.golang.migration.MigrateXExpSlicesToStdlib"
}
func (r *MigrateXExpSlicesToStdlib) DisplayName() string {
	return "Migrate `golang.org/x/exp/slices` to `slices`"
}
func (r *MigrateXExpSlicesToStdlib) Description() string {
	return "Repoint `golang.org/x/exp/slices` at the standard library `slices`, added in Go 1.21. Every x/exp function exists there under the same name and signature, so call sites are unchanged. A file is skipped when the module targets an older Go release, when it already imports `slices`, or when it passes a boolean comparator to `SortFunc`, `SortStableFunc`, `IsSortedFunc`, `MinFunc` or `MaxFunc` — the pre-2023 x/exp signature, which needs a hand conversion to a three-way `cmp` function."
}
func (r *MigrateXExpSlicesToStdlib) Tags() []string {
	return []string{"migration", "slices", "stdlib"}
}

func (r *MigrateXExpSlicesToStdlib) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "exptostd", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *MigrateXExpSlicesToStdlib) InitialValue(*recipe.ExecutionContext) any {
	return newModuleAcc()
}
func (r *MigrateXExpSlicesToStdlib) Scanner(acc any) recipe.TreeVisitor {
	return scanGoVersion(acc.(*moduleAcc))
}
func (r *MigrateXExpSlicesToStdlib) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&migrateSlicesVisitor{acc: acc.(*moduleAcc)})
}

type migrateSlicesVisitor struct {
	visitor.GoVisitor
	acc *moduleAcc
}

func (v *migrateSlicesVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	pkg := pathswap.Qualifier(cu, expSlices)
	if pkg == "" || !v.acc.atLeast(go121) {
		return cu
	}
	// Both imports would bind the same package name, and the standard library
	// one already covers the file's usage.
	if pathswap.Imports(cu, stdSlices) {
		return cu
	}
	if usesLegacyLessFunc(cu, pkg) {
		return cu
	}

	swapped := pathswap.RewriteImports(cu, slicesRules)
	if swapped == cu {
		return cu
	}
	if retyped, ok := pathswap.Retype(slicesRules).Visit(swapped, p).(*golang.CompilationUnit); ok {
		return retyped
	}
	return swapped
}

// usesLegacyLessFunc reports whether cu calls one of the reshaped x/exp/slices
// functions with a boolean-returning function literal — the comparator the
// package took before February 2023, which neither the current x/exp nor the
// standard library accepts.
func usesLegacyLessFunc(cu *golang.CompilationUnit, pkg string) bool {
	scan := visitor.Init(&legacyLessFuncScan{pkg: pkg})
	scan.Visit(cu, nil)
	return scan.found
}

type legacyLessFuncScan struct {
	visitor.GoVisitor
	pkg   string
	found bool
}

func (s *legacyLessFuncScan) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	name, ok := qualifiedCall(mi, s.pkg)
	if ok && lessFuncTakers[name] {
		for _, arg := range realArgs(mi) {
			if returnsBool(arg) {
				s.found = true
			}
		}
	}
	return s.GoVisitor.VisitMethodInvocation(mi, p)
}

// returnsBool reports whether expr is a function literal declaring a single bool
// result. A comparator named rather than spelled out cannot be read here, and is
// assumed to be the current three-way form.
func returnsBool(expr java.Expression) bool {
	lit, ok := funcLiteral(expr)
	if !ok || lit.ReturnType == nil {
		return false
	}
	id, ok := lit.ReturnType.(*java.Identifier)
	return ok && id.Name == "bool"
}

// funcLiteral unwraps a function literal in expression position. A
// java.MethodDeclaration is a statement, so the parser wraps one used as an
// argument in a golang.StatementExpression; a type assertion straight to
// *java.MethodDeclaration misses every such literal.
func funcLiteral(expr java.Expression) (*java.MethodDeclaration, bool) {
	if se, ok := expr.(*golang.StatementExpression); ok {
		md, ok := se.Statement.(*java.MethodDeclaration)
		return md, ok
	}
	md, ok := expr.(*java.MethodDeclaration)
	return md, ok
}
