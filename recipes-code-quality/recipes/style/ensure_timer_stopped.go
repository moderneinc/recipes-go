/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// EnsureTimerStopped finds calls to `time.NewTimer` and `time.AfterFunc`
// and inserts `defer timer.Stop()` after the assignment.
type EnsureTimerStopped struct {
	recipe.Base
}

func (r *EnsureTimerStopped) Name() string {
	return "org.openrewrite.golang.codequality.EnsureTimerStopped"
}
func (r *EnsureTimerStopped) DisplayName() string { return "Ensure timer stopped" }
func (r *EnsureTimerStopped) Description() string {
	return "Find calls to `time.NewTimer` and `time.AfterFunc`. Timers should be stopped when no longer needed to release resources."
}
func (r *EnsureTimerStopped) Tags() []string { return []string{"style", "resource-management"} }

func (r *EnsureTimerStopped) Editor() recipe.TreeVisitor {
	return visitor.Init(&ensureTimerStoppedVisitor{})
}

type ensureTimerStoppedVisitor struct {
	visitor.GoVisitor
}

var timerMethods = map[string]bool{
	"NewTimer":  true,
	"AfterFunc": true,
}

func (v *ensureTimerStoppedVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	var newStmts []java.RightPadded[java.Statement]
	changed := false

	for i, rp := range block.Statements {
		newStmts = append(newStmts, rp)

		if varName, ok := extractAssignedVar(rp.Element, isTimeTimer); ok {
			if hasDeferAfter(block.Statements, i, varName, "Stop") {
				continue
			}
			deferStmt := buildDeferMethodCall(varName, "Stop", rp.Element)
			newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: deferStmt})
			changed = true
		}
	}

	if changed {
		return block.WithStatements(newStmts)
	}
	return block
}

// isTimeTimer returns true if the method invocation is time.NewTimer or time.AfterFunc.
func isTimeTimer(mi *java.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "time" {
		return false
	}
	return timerMethods[mi.Name.Name]
}
