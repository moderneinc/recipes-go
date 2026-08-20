/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AdoptTestifyRequireLen replaces `if len(x) != n { t.Fatal(...) }` with
// `require.Len(t, x, n)`.
type AdoptTestifyRequireLen struct {
	recipe.Base
}

func (r *AdoptTestifyRequireLen) Name() string {
	return "org.openrewrite.golang.testify.AdoptTestifyRequireLen"
}
func (r *AdoptTestifyRequireLen) DisplayName() string { return "Adopt testify require.Len" }
func (r *AdoptTestifyRequireLen) Description() string {
	return "Replace `if len(x) != n { t.Fatal(...) }` with `require.Len(t, x, n)` from `github.com/stretchr/testify/require`."
}
func (r *AdoptTestifyRequireLen) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestifyRequireLen) Editor() recipe.TreeVisitor {
	return visitor.Init(&lenVisitor{guardBase{pkg: requirePkg, importPath: requireImport, isReporter: isFatalReporter}})
}

// AdoptTestifyAssertLen replaces `if len(x) != n { t.Error(...) }` with
// `assert.Len(t, x, n)`.
type AdoptTestifyAssertLen struct {
	recipe.Base
}

func (r *AdoptTestifyAssertLen) Name() string {
	return "org.openrewrite.golang.testify.AdoptTestifyAssertLen"
}
func (r *AdoptTestifyAssertLen) DisplayName() string { return "Adopt testify assert.Len" }
func (r *AdoptTestifyAssertLen) Description() string {
	return "Replace `if len(x) != n { t.Error(...) }` with `assert.Len(t, x, n)` from `github.com/stretchr/testify/assert`."
}
func (r *AdoptTestifyAssertLen) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestifyAssertLen) Editor() recipe.TreeVisitor {
	return visitor.Init(&lenVisitor{guardBase{pkg: assertPkg, importPath: assertImport, isReporter: isErrorReporter}})
}

type lenVisitor struct {
	guardBase
}

func (v *lenVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)
	return rewriteBlockGuards(v, block, v.importPath, v.matchLen)
}

// matchLen returns the `<pkg>.Len(t, x, n)` replacement for `if len(x) != n { ... }`,
// or nil. Only `!=` maps to Len; `==` asserts a non-length, which Len cannot
// express.
func (v *lenVisitor) matchLen(ifStmt *java.If) *java.MethodInvocation {
	if ifStmt.Condition == nil || ifStmt.ElsePart != nil {
		return nil
	}
	bin, ok := ifStmt.Condition.Tree.Element.(*java.Binary)
	if !ok || bin.Operator.Element != java.NotEqual {
		return nil
	}

	collection, count := lenOperands(bin.Left, bin.Right)
	if collection == nil {
		return nil
	}

	call, recv := reporterBodyCall(ifStmt.ThenPart.Element, v.isReporter)
	if call == nil {
		return nil
	}
	return finishAssertion(ifStmt.GetPrefix(), v.pkg, v.importPath, recv, "Len", call, []java.Expression{collection, count}, identSet(collection, count), nil)
}

// lenOperands returns (collection, count) when exactly one of left/right is a
// `len(collection)` call and the other is the expected count, or (nil, nil).
func lenOperands(left, right java.Expression) (collection, count java.Expression) {
	lc, rc := lenArg(left), lenArg(right)
	switch {
	case lc != nil && rc == nil:
		return lc, right
	case rc != nil && lc == nil:
		return rc, left
	}
	return nil, nil
}

// lenArg returns the argument of a builtin `len(x)` call, or nil.
func lenArg(expr java.Expression) java.Expression {
	mi, ok := expr.(*java.MethodInvocation)
	if !ok || mi.Select != nil || mi.Name == nil || mi.Name.Name != "len" {
		return nil
	}
	if len(mi.Arguments.Elements) != 1 {
		return nil
	}
	return mi.Arguments.Elements[0].Element
}
