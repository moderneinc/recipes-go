/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package expstd

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

var slogRules = []pathswap.Rule{{Old: expSlog, New: stdSlog}}

// Migrates golang.org/x/exp/slog to log/slog.
type MigrateXExpSlogToStdlib struct {
	recipe.ScanningBase
}

func (r *MigrateXExpSlogToStdlib) Name() string {
	return "org.openrewrite.golang.migration.MigrateXExpSlogToStdlib"
}
func (r *MigrateXExpSlogToStdlib) DisplayName() string {
	return "Migrate `golang.org/x/exp/slog` to `log/slog`"
}
func (r *MigrateXExpSlogToStdlib) Description() string {
	return "Repoint `golang.org/x/exp/slog` at the standard library `log/slog`, added in Go 1.21, and rename the context-taking helpers the standard library spells differently: `DebugCtx`, `InfoCtx`, `WarnCtx` and `ErrorCtx` become `DebugContext`, `InfoContext`, `WarnContext` and `ErrorContext`. x/exp carries both spellings, so a file already on the `Context` ones is a plain path swap."
}
func (r *MigrateXExpSlogToStdlib) Tags() []string {
	return []string{"migration", "slog", "logging", "stdlib"}
}

func (r *MigrateXExpSlogToStdlib) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "exptostd", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *MigrateXExpSlogToStdlib) InitialValue(*recipe.ExecutionContext) any {
	return newModuleAcc()
}
func (r *MigrateXExpSlogToStdlib) Scanner(acc any) recipe.TreeVisitor {
	return scanGoVersion(acc.(*moduleAcc))
}
func (r *MigrateXExpSlogToStdlib) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&migrateSlogVisitor{acc: acc.(*moduleAcc)})
}

type migrateSlogVisitor struct {
	visitor.GoVisitor
	acc *moduleAcc
	pkg string
}

func (v *migrateSlogVisitor) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	pkg := pathswap.Qualifier(cu, expSlog)
	if pkg == "" || !v.acc.atLeast(go121) {
		return cu
	}
	// Both imports bind the same name, and the standard library one already
	// covers the file's usage.
	if pathswap.Imports(cu, stdSlog) {
		return cu
	}

	v.pkg = pkg
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)

	swapped := pathswap.RewriteImports(cu, slogRules)
	if retyped, ok := pathswap.Retype(slogRules).Visit(swapped, p).(*golang.CompilationUnit); ok {
		return retyped
	}
	return swapped
}

func (v *migrateSlogVisitor) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	mi = v.GoVisitor.VisitMethodInvocation(mi, p).(*java.MethodInvocation)
	if mi.Name == nil {
		return mi
	}
	renamed, ok := slogCtxRenames[mi.Name.Name]
	if !ok {
		return mi
	}
	onPackage, isSlogCall := v.slogCtxCallShape(mi)
	if !isSlogCall {
		return mi
	}

	// The parser types these calls once the import is log/slog, so the renamed
	// one has to carry the signature a parsed one would. Both helpers return
	// nothing.
	declaring := stdSlog + ".Logger"
	if onPackage {
		declaring = stdSlog
	}
	c := *mi
	c.Name = &java.Identifier{Prefix: mi.Name.Prefix, Name: renamed, Type: mi.Name.Type}
	c.MethodType = lstutil.FuncType(declaring, renamed, nil)
	return &c
}

// slogCtxCallShape reports whether mi is one of the renamed helpers, and whether
// it is called on the slog package rather than on a logger value. A logger
// receiver's type cannot confirm it where x/exp is off the parse classpath, so a
// leading context.Context argument stands in.
func (v *migrateSlogVisitor) slogCtxCallShape(mi *java.MethodInvocation) (onPackage, isSlogCall bool) {
	if _, ok := qualifiedCall(mi, v.pkg); ok {
		return true, true
	}
	args := realArgs(mi)
	if len(args) == 0 {
		return false, false
	}
	return false, matcher.GetFullyQualifiedName(matcher.TypeOfExpression(args[0])) == "context.Context"
}
