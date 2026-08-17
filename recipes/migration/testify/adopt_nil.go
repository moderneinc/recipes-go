/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AdoptTestifyRequireNil replaces `if x != nil { t.Fatal(...) }` with
// `require.Nil(t, x)` and `if x == nil { t.Fatal(...) }` with
// `require.NotNil(t, x)`, for non-error operands (errors are handled by the
// NoError / Error recipes).
type AdoptTestifyRequireNil struct {
	recipe.Base
}

func (r *AdoptTestifyRequireNil) Name() string {
	return "org.openrewrite.golang.testify.AdoptTestifyRequireNil"
}
func (r *AdoptTestifyRequireNil) DisplayName() string { return "Adopt testify require.Nil" }
func (r *AdoptTestifyRequireNil) Description() string {
	return "Replace `if x != nil { t.Fatal(...) }` with `require.Nil(t, x)` and `if x == nil { t.Fatal(...) }` with `require.NotNil(t, x)` from `github.com/stretchr/testify/require`, for non-error operands. Error operands are handled by the NoError / Error recipes."
}
func (r *AdoptTestifyRequireNil) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestifyRequireNil) Editor() recipe.TreeVisitor {
	return visitor.Init(&nilVisitor{guardBase{pkg: requirePkg, importPath: requireImport, isReporter: isFatalReporter}})
}

// AdoptTestifyAssertNil replaces `if x != nil { t.Error(...) }` with
// `assert.Nil(t, x)` and `if x == nil { t.Error(...) }` with
// `assert.NotNil(t, x)`, for non-error operands.
type AdoptTestifyAssertNil struct {
	recipe.Base
}

func (r *AdoptTestifyAssertNil) Name() string {
	return "org.openrewrite.golang.testify.AdoptTestifyAssertNil"
}
func (r *AdoptTestifyAssertNil) DisplayName() string { return "Adopt testify assert.Nil" }
func (r *AdoptTestifyAssertNil) Description() string {
	return "Replace `if x != nil { t.Error(...) }` with `assert.Nil(t, x)` and `if x == nil { t.Error(...) }` with `assert.NotNil(t, x)` from `github.com/stretchr/testify/assert`, for non-error operands. Error operands are handled by the NoError / Error recipes."
}
func (r *AdoptTestifyAssertNil) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestifyAssertNil) Editor() recipe.TreeVisitor {
	return visitor.Init(&nilVisitor{guardBase{pkg: assertPkg, importPath: assertImport, isReporter: isErrorReporter}})
}

type nilVisitor struct {
	guardBase
}

func (v *nilVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	return rewriteBlockGuards(v, block, v.importPath, v.matchNil)
}

// matchNil returns the Nil/NotNil replacement for a non-error nil comparison, or
// nil. `if x != nil` asserts x is nil; `if x == nil` asserts x is not nil.
func (v *nilVisitor) matchNil(ifStmt *java.If) *java.MethodInvocation {
	if ifStmt.Condition == nil || ifStmt.ElsePart != nil {
		return nil
	}
	bin, ok := ifStmt.Condition.Tree.Element.(*java.Binary)
	if !ok {
		return nil
	}

	var assertion string
	switch bin.Operator.Element {
	case java.NotEqual:
		assertion = "Nil"
	case java.Equal:
		assertion = "NotNil"
	default:
		return nil
	}

	var operand java.Expression
	switch {
	case isNilIdent(bin.Right) && !isNilIdent(bin.Left):
		operand = bin.Left
	case isNilIdent(bin.Left) && !isNilIdent(bin.Right):
		operand = bin.Right
	default:
		return nil
	}
	// Error operands belong to the NoError / Error recipes.
	if looksLikeError(operand) {
		return nil
	}

	call, recv := reporterBodyCall(ifStmt.ThenPart.Element, v.isReporter)
	if call == nil {
		return nil
	}
	return finishAssertion(ifStmt.GetPrefix(), v.pkg, recv.Name, assertion, call, []java.Expression{operand}, identSet(operand), nil)
}
