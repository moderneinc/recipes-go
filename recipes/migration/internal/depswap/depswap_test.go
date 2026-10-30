/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package depswap

import (
	"testing"

	"github.com/moderneinc/recipes-go/recipes/migration/internal/pathswap"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/parser"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var swapModules = Modules{Old: "example.com/old", New: []string{"example.com/new"}, Version: "v1.0.0"}

type deferredSwap struct {
	visitor.GoVisitor
}

func (v *deferredSwap) VisitCompilationUnit(cu *golang.CompilationUnit, p any) java.J {
	v.DoAfterVisit(swapImports{})
	return cu
}

type swapImports struct{}

func (swapImports) Visit(t java.Tree, p any) java.Tree {
	cu, ok := t.(*golang.CompilationUnit)
	if !ok {
		return t
	}
	return pathswap.RewriteImports(cu, []pathswap.Rule{{Old: swapModules.Old, New: swapModules.New[0]}})
}

func TestScannerSeesImportsAnEditorAddsInAnAfterVisit(t *testing.T) {
	// given
	cu, err := parser.NewGoParser().Parse("app/app.go", "package app\n\nimport \"example.com/old\"\n\nvar _ = old.X\n")
	require.NoError(t, err)
	acc := NewAcc()
	scanner := Scanner(acc, swapModules, EditorProbe(func() recipe.TreeVisitor {
		return visitor.Init(&deferredSwap{})
	}))

	// when
	scanner.Visit(cu, recipe.NewExecutionContext())

	// then
	assert.Equal(t, &Acc{NeedsNew: true}, acc)
}
