/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// EnsureTickerStopped finds calls to `time.NewTicker` and inserts
// `defer ticker.Stop()` after the assignment.
type EnsureTickerStopped struct {
	recipe.Base
}

func (r *EnsureTickerStopped) Name() string {
	return "org.openrewrite.golang.codequality.EnsureTickerStopped"
}
func (r *EnsureTickerStopped) DisplayName() string { return "Ensure ticker stopped" }
func (r *EnsureTickerStopped) Description() string {
	return "Find calls to `time.NewTicker`. Tickers must be stopped when no longer needed to avoid goroutine leaks."
}
func (r *EnsureTickerStopped) Tags() []string { return []string{"style", "resource-management"} }

func (r *EnsureTickerStopped) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureTickerStoppedVisitor{})
}

type ensureTickerStoppedVisitor struct {
	visitor.GoVisitor
}

func (v *ensureTickerStoppedVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	var newStmts []tree.RightPadded[tree.Statement]
	changed := false

	for i, rp := range block.Statements {
		newStmts = append(newStmts, rp)

		if varName, ok := extractAssignedVar(rp.Element, isTimeNewTicker); ok {
			if hasDeferAfter(block.Statements, i, varName, "Stop") {
				continue
			}
			deferStmt := buildDeferMethodCall(varName, "Stop", rp.Element)
			newStmts = append(newStmts, tree.RightPadded[tree.Statement]{Element: deferStmt})
			changed = true
		}
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// isTimeNewTicker returns true if the method invocation is time.NewTicker.
func isTimeNewTicker(mi *tree.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*tree.Identifier)
	if !ok || ident.Name != "time" {
		return false
	}
	return mi.Name.Name == "NewTicker"
}
