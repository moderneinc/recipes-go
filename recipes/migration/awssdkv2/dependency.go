/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package awssdkv2

import (
	"go/version"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// pinnedModules are the v2 modules this recipe knows a version for. A service
// module is versioned on its own schedule, so `go mod tidy` adds those.
var pinnedModules = map[string]string{
	v2Module:      v2CoreVersion,
	v2Config:      v2ConfigVersion,
	v2Credentials: v2CredentialsVersion,
	smithyGo:      smithyVersion,
}

// Brings the go.mod requires into line with what the migrated source imports.
type UpdateAwsSdkDependency struct {
	recipe.ScanningBase
}

func (r *UpdateAwsSdkDependency) Name() string {
	return "org.openrewrite.golang.migration.UpdateAwsSdkDependency"
}
func (r *UpdateAwsSdkDependency) DisplayName() string {
	return "Require `aws-sdk-go-v2` instead of `aws-sdk-go`"
}
func (r *UpdateAwsSdkDependency) Description() string {
	return "Require the `github.com/aws/aws-sdk-go-v2` modules the migrated source imports, and drop `github.com/aws/aws-sdk-go` once no file imports it. Only the core, `config` and `credentials` modules are pinned here; every service is its own independently versioned module, so `go mod tidy` adds those from the imports. Does not sync go.sum."
}
func (r *UpdateAwsSdkDependency) Tags() []string {
	return []string{"migration", "aws", "gomod"}
}

func (r *UpdateAwsSdkDependency) InitialValue(*recipe.ExecutionContext) any {
	return &awsUsageAcc{needed: map[string]bool{}}
}
func (r *UpdateAwsSdkDependency) Scanner(acc any) recipe.TreeVisitor {
	return visitor.Init(&awsUsageScanner{acc: acc.(*awsUsageAcc)})
}
func (r *UpdateAwsSdkDependency) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&awsRequireEditor{acc: acc.(*awsUsageAcc)})
}

type awsUsageAcc struct {
	// needed holds the pinned v2 modules the source will import.
	needed map[string]bool
	// v1Remains is set by a file that stays on v1.
	v1Remains bool
	// migrated is set by a file that moves to v2. A module importing only
	// service packages needs no pinned require — those are left to
	// `go mod tidy` — but still needs the v1 require dropped and the `go`
	// directive raised.
	migrated bool
}

type awsUsageScanner struct {
	visitor.GoVisitor
	acc *awsUsageAcc
}

// The scan asks where each file ends up rather than where it is, so the answer
// does not depend on whether the source rewrite has run yet — the Moderne CLI
// scans before a recipe list's edits are applied.
func (v *awsUsageScanner) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	recordAwsUsage(v.acc, cu)
	return cu
}

// recordAwsUsage notes which requires a file will need once migrated. The scan
// asks where each file ends up rather than where it is, so the answer does not
// depend on whether the source rewrite has run yet.
func recordAwsUsage(acc *awsUsageAcc, cu *golang.CompilationUnit) {
	if cu.Imports == nil || vendored(cu) {
		return
	}
	for _, rp := range cu.Imports.Elements {
		if module, ok := pinnedModuleFor(pathswap.Path(rp.Element)); ok {
			acc.needed[module] = true
		}
	}
	if !importsV1(cu) {
		return
	}
	if !scanFile(cu).migratable {
		acc.v1Remains = true
		return
	}
	acc.migrated = true
	for _, rp := range cu.Imports.Elements {
		newPath, ok := MapPath(pathswap.Path(rp.Element))
		if !ok {
			continue
		}
		if module, ok := pinnedModuleFor(newPath); ok {
			acc.needed[module] = true
		}
	}
}

// pinnedModuleFor returns the v2 module providing an import path, when that
// module is one this recipe pins.
func pinnedModuleFor(path string) (string, bool) {
	switch path {
	case v2Aws:
		return v2Module, true
	case v2Config:
		return v2Config, true
	case v2Credentials:
		return v2Credentials, true
	case smithyGo:
		return smithyGo, true
	}
	return "", false
}

type awsRequireEditor struct {
	visitor.GoVisitor
	acc *awsUsageAcc
}

func (v *awsRequireEditor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	return applyAwsRequires(gm, v.acc)
}

// applyAwsRequires brings the go.mod requires into line with what the migrated
// source imports.
func applyAwsRequires(gm *golang.GoMod, acc *awsUsageAcc) java.Tree {
	if !acc.migrated && len(acc.needed) == 0 {
		return gm
	}
	// Added in module-path order so the block does not depend on the order the
	// scan happened to see files in.
	for _, module := range []string{v2Module, v2Config, v2Credentials, smithyGo} {
		if acc.needed[module] {
			gm = migration.AddRequire(gm, module, pinnedModules[module], false)
		}
	}
	if !acc.v1Remains {
		gm = migration.RemoveRequire(gm, v1Module)
	}
	return raiseGoDirective(gm, v2MinGo)
}

// raiseGoDirective lifts the `go` directive to the release aws-sdk-go-v2
// requires, since the module cannot build against v2 below it.
func raiseGoDirective(gm *golang.GoMod, minVersion string) *golang.GoMod {
	statements := make([]java.RightPadded[golang.GoModStatement], len(gm.Statements))
	copy(statements, gm.Statements)
	changed := false
	for i, rp := range statements {
		d, ok := rp.Element.(*golang.GoModDirective)
		if !ok || d.Keyword != "go" || len(d.Values) != 1 {
			continue
		}
		if version.Compare("go"+d.Values[0].Text, "go"+minVersion) >= 0 {
			continue
		}
		values := []*golang.GoModValue{d.Values[0].WithText(minVersion)}
		statements[i].Element = d.WithValues(values)
		changed = true
	}
	if !changed {
		return gm
	}
	return gm.WithStatements(statements)
}
