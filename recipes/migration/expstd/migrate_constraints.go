/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package expstd

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var constraintsRules = []pathswap.Rule{{Old: expConstraints, New: stdCmp}}

// Migrates golang.org/x/exp/constraints.Ordered to cmp.Ordered.
type MigrateXExpConstraintsToStdlib struct {
	recipe.ScanningBase
}

func (r *MigrateXExpConstraintsToStdlib) Name() string {
	return "org.openrewrite.golang.migration.MigrateXExpConstraintsToStdlib"
}
func (r *MigrateXExpConstraintsToStdlib) DisplayName() string {
	return "Migrate `golang.org/x/exp/constraints` to `cmp`"
}
func (r *MigrateXExpConstraintsToStdlib) Description() string {
	return "Rewrite `constraints.Ordered` to `cmp.Ordered`, added to the standard library in Go 1.21. `Integer`, `Float`, `Signed`, `Unsigned` and `Complex` have no standard-library counterpart, so a file naming one of them keeps the x/exp import and is left for review."
}
func (r *MigrateXExpConstraintsToStdlib) Tags() []string {
	return []string{"migration", "constraints", "stdlib"}
}

func (r *MigrateXExpConstraintsToStdlib) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "exptostd", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *MigrateXExpConstraintsToStdlib) InitialValue(*recipe.ExecutionContext) any {
	return newModuleAcc()
}
func (r *MigrateXExpConstraintsToStdlib) Scanner(acc any) recipe.TreeVisitor {
	return scanGoVersion(acc.(*moduleAcc))
}
func (r *MigrateXExpConstraintsToStdlib) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&migrateConstraintsVisitor{acc: acc.(*moduleAcc)})
}

type migrateConstraintsVisitor struct {
	visitor.GoVisitor
	acc *moduleAcc
	// pkg is the file's local name for x/exp/constraints; renameQualifier records
	// that the swap changes that name, which an aliased import does not.
	pkg             string
	renameQualifier bool
}

// The types the rewritten references carry, so that RemoveUnusedImports and any
// later type-based match read them as the standard library's.
var (
	cmpPackageType = lstutil.NamedType(stdCmp)
	cmpOrderedType = lstutil.NamedType(stdCmp + ".Ordered")
)

func (v *migrateConstraintsVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	imp := pathswap.Find(cu, expConstraints)
	pkg := pathswap.Qualifier(cu, expConstraints)
	if pkg == "" || !v.acc.atLeast(go121) {
		return cu
	}
	if usesUnportedConstraint(cu, pkg) {
		return cu
	}
	// An aliased import keeps its name across the swap, so only an unaliased one
	// renames its references — and only then can the emitted `cmp` collide.
	v.renameQualifier = pathswap.Alias(imp) == ""
	if v.renameQualifier && bindsName(cu, stdCmp) {
		return cu
	}

	v.pkg = pkg
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)

	swapped := pathswap.RewriteImports(cu, constraintsRules)
	if retyped, ok := pathswap.Retype(constraintsRules).Visit(swapped, p).(*golang.CompilationUnit); ok {
		return retyped
	}
	return swapped
}

func (v *migrateConstraintsVisitor) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	fa = v.GoVisitor.VisitFieldAccess(fa, p).(*java.FieldAccess)
	name, ok := qualifiedRef(fa, v.pkg)
	if !ok || name != "Ordered" {
		return fa
	}

	c := *fa
	if v.renameQualifier {
		target := fa.Target.(*java.Identifier)
		c.Target = &java.Identifier{Prefix: target.Prefix, Name: stdCmp, Type: cmpPackageType}
	}
	c.Type = cmpOrderedType
	return &c
}

// usesUnportedConstraint reports whether cu names an x/exp constraint the
// standard library never adopted, which would be stranded by the swap.
func usesUnportedConstraint(cu *golang.CompilationUnit, pkg string) bool {
	scan := visitor.Init(&unportedConstraintScan{pkg: pkg})
	scan.Visit(cu, nil)
	return scan.found
}

type unportedConstraintScan struct {
	visitor.GoVisitor
	pkg   string
	found bool
}

func (s *unportedConstraintScan) VisitFieldAccess(fa *java.FieldAccess, p any) java.J {
	if name, ok := qualifiedRef(fa, s.pkg); ok && expOnlyConstraints[name] {
		s.found = true
	}
	return s.GoVisitor.VisitFieldAccess(fa, p)
}
