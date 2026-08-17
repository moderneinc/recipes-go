/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify

import (
	"strings"

	"github.com/moderneinc/recipes-go/recipes/migration"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

const (
	testifyModulePath = "github.com/stretchr/testify"
	testifyVersion    = "v1.10.0"
)

// AddTestifyDependency adds a `require github.com/stretchr/testify` directive to
// go.mod when a package in the module imports testify but go.mod does not yet
// require it — the go.mod counterpart of the source-level adoption recipes.
//
// It is a scanning recipe: the scan phase records whether any .go file imports
// testify, then the edit phase adds the require to go.mod. It does not sync
// go.sum or add testify's own indirect dependencies (those need module hashes
// that cannot be computed offline), so a `go mod tidy` / `go mod download` is
// still needed to complete resolution.
type AddTestifyDependency struct {
	recipe.ScanningBase
}

func (r *AddTestifyDependency) Name() string {
	return "org.openrewrite.golang.testify.AddTestifyDependency"
}
func (r *AddTestifyDependency) DisplayName() string {
	return "Add the testify dependency to go.mod"
}
func (r *AddTestifyDependency) Description() string {
	return "Add a `require github.com/stretchr/testify` directive to go.mod when a package in the module imports testify but go.mod does not yet require it. Does not sync go.sum or add transitive dependencies; a `go mod tidy` is still needed to complete resolution."
}
func (r *AddTestifyDependency) Tags() []string { return []string{"testing", "testify", "gomod"} }

func (r *AddTestifyDependency) InitialValue(ctx *recipe.ExecutionContext) any {
	return &testifyUsageAcc{}
}

func (r *AddTestifyDependency) Scanner(acc any) recipe.TreeVisitor {
	return visitor.Init(&testifyImportScanner{acc: acc.(*testifyUsageAcc)})
}

func (r *AddTestifyDependency) EditorWithData(acc any) recipe.TreeVisitor {
	return visitor.Init(&addTestifyRequireEditor{acc: acc.(*testifyUsageAcc)})
}

type testifyUsageAcc struct {
	imported bool
}

type testifyImportScanner struct {
	visitor.GoVisitor
	acc *testifyUsageAcc
}

func (v *testifyImportScanner) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	cu = v.GoVisitor.VisitCompilationUnit(cu, p).(*golang.CompilationUnit)
	if v.acc.imported || cu.Imports == nil {
		return cu
	}
	for _, rp := range cu.Imports.Elements {
		if importsTestify(importPathOf(rp.Element)) {
			v.acc.imported = true
			break
		}
	}
	return cu
}

type addTestifyRequireEditor struct {
	visitor.GoVisitor
	acc *testifyUsageAcc
}

func (v *addTestifyRequireEditor) VisitGoMod(gm *golang.GoMod, p any) java.Tree {
	if !v.acc.imported {
		return gm
	}
	return migration.AddRequire(gm, testifyModulePath, testifyVersion, false)
}

func importsTestify(importPath string) bool {
	return importPath == testifyModulePath || strings.HasPrefix(importPath, testifyModulePath+"/")
}

// importPathOf returns the unquoted import path of an import spec, or "" when the
// spec is not a plain string literal.
func importPathOf(imp *java.Import) string {
	if imp == nil {
		return ""
	}
	lit, ok := imp.Qualid.(*java.Literal)
	if !ok || lit == nil {
		return ""
	}
	raw := lit.Source
	if s, ok := lit.Value.(string); ok {
		raw = s
	}
	return strings.Trim(raw, "\"`")
}
