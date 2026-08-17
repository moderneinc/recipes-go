/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AdoptTestifyRequireTrue replaces `if !cond { t.Fatal(...) }` with
// `require.True(t, cond)` and `if cond { t.Fatal(...) }` with
// `require.False(t, cond)`. Conditions that are comparisons are left to the more
// specific Equal / Nil / Len recipes.
type AdoptTestifyRequireTrue struct {
	recipe.Base
}

func (r *AdoptTestifyRequireTrue) Name() string {
	return "org.openrewrite.golang.testify.AdoptTestifyRequireTrue"
}
func (r *AdoptTestifyRequireTrue) DisplayName() string { return "Adopt testify require.True" }
func (r *AdoptTestifyRequireTrue) Description() string {
	return "Replace `if !cond { t.Fatal(...) }` with `require.True(t, cond)` and `if cond { t.Fatal(...) }` with `require.False(t, cond)` from `github.com/stretchr/testify/require`. Comparison conditions are left to the Equal / Nil / Len recipes."
}
func (r *AdoptTestifyRequireTrue) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestifyRequireTrue) Editor() recipe.TreeVisitor {
	return visitor.Init(&boolVisitor{guardBase{pkg: requirePkg, importPath: requireImport, isReporter: isFatalReporter}})
}

// AdoptTestifyAssertTrue replaces `if !cond { t.Error(...) }` with
// `assert.True(t, cond)` and `if cond { t.Error(...) }` with
// `assert.False(t, cond)`.
type AdoptTestifyAssertTrue struct {
	recipe.Base
}

func (r *AdoptTestifyAssertTrue) Name() string {
	return "org.openrewrite.golang.testify.AdoptTestifyAssertTrue"
}
func (r *AdoptTestifyAssertTrue) DisplayName() string { return "Adopt testify assert.True" }
func (r *AdoptTestifyAssertTrue) Description() string {
	return "Replace `if !cond { t.Error(...) }` with `assert.True(t, cond)` and `if cond { t.Error(...) }` with `assert.False(t, cond)` from `github.com/stretchr/testify/assert`. Comparison conditions are left to the Equal / Nil / Len recipes."
}
func (r *AdoptTestifyAssertTrue) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestifyAssertTrue) Editor() recipe.TreeVisitor {
	return visitor.Init(&boolVisitor{guardBase{pkg: assertPkg, importPath: assertImport, isReporter: isErrorReporter}})
}

type boolVisitor struct {
	guardBase
}

func (v *boolVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	return rewriteBlockGuards(v, block, v.importPath, v.matchBool)
}

// matchBool returns the True/False replacement for a boolean guard, or nil.
// `if !cond` asserts cond is true; `if cond` asserts cond is false.
func (v *boolVisitor) matchBool(ifStmt *java.If) *java.MethodInvocation {
	if ifStmt.Condition == nil || ifStmt.ElsePart != nil {
		return nil
	}
	cond := ifStmt.Condition.Tree.Element

	var assertion string
	var boolExpr java.Expression
	if u, ok := cond.(*java.Unary); ok && u.Operator.Element == java.Not {
		assertion, boolExpr = "True", u.Operand
	} else if isComparison(cond) {
		return nil
	} else {
		assertion, boolExpr = "False", cond
	}

	call, recv := reporterBodyCall(ifStmt.ThenPart.Element, v.isReporter)
	if call == nil {
		return nil
	}
	return finishAssertion(ifStmt.GetPrefix(), v.pkg, recv.Name, assertion, call, []java.Expression{boolExpr}, identSet(boolExpr), nil)
}
