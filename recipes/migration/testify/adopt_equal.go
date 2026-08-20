/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AdoptTestifyRequireEqual replaces `if got != want { t.Fatal(...) }` /
// `if got == want { t.Fatal(...) }` comparison guards in tests with
// `require.Equal(t, want, got)` / `require.NotEqual(t, want, got)`.
type AdoptTestifyRequireEqual struct {
	recipe.Base
}

func (r *AdoptTestifyRequireEqual) Name() string {
	return "org.openrewrite.golang.testify.AdoptTestifyRequireEqual"
}
func (r *AdoptTestifyRequireEqual) DisplayName() string {
	return "Adopt testify require.Equal"
}
func (r *AdoptTestifyRequireEqual) Description() string {
	return "Replace `if got != want { t.Fatal(...) }` and `if got == want { t.Fatal(...) }` comparison guards in tests with `require.Equal(t, want, got)` / `require.NotEqual(t, want, got)` from `github.com/stretchr/testify/require`."
}
func (r *AdoptTestifyRequireEqual) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestifyRequireEqual) Editor() recipe.TreeVisitor {
	return visitor.Init(&equalityVisitor{pkg: requirePkg, importPath: requireImport, isReporter: isFatalReporter})
}

// AdoptTestifyAssertEqual replaces `if got != want { t.Error(...) }` /
// `if got == want { t.Error(...) }` comparison guards in tests with
// `assert.Equal(t, want, got)` / `assert.NotEqual(t, want, got)`.
type AdoptTestifyAssertEqual struct {
	recipe.Base
}

func (r *AdoptTestifyAssertEqual) Name() string {
	return "org.openrewrite.golang.testify.AdoptTestifyAssertEqual"
}
func (r *AdoptTestifyAssertEqual) DisplayName() string {
	return "Adopt testify assert.Equal"
}
func (r *AdoptTestifyAssertEqual) Description() string {
	return "Replace `if got != want { t.Error(...) }` and `if got == want { t.Error(...) }` comparison guards in tests with `assert.Equal(t, want, got)` / `assert.NotEqual(t, want, got)` from `github.com/stretchr/testify/assert`."
}
func (r *AdoptTestifyAssertEqual) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestifyAssertEqual) Editor() recipe.TreeVisitor {
	return visitor.Init(&equalityVisitor{pkg: assertPkg, importPath: assertImport, isReporter: isErrorReporter})
}

// equalityVisitor rewrites `if a <op> b { <reporter>(...) }` comparison guards to
// `<pkg>.Equal`/`<pkg>.NotEqual(t, expected, actual)`. `!=` asserts equality,
// `==` asserts inequality; the reporter selects require vs assert.
type equalityVisitor struct {
	visitor.GoVisitor
	pkg        string
	importPath string
	isReporter func(string) bool
}

func (v *equalityVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	changed := false
	newStmts := make([]java.RightPadded[java.Statement], 0, len(block.Statements))
	for _, rp := range block.Statements {
		var call *java.MethodInvocation
		if ifStmt, ok := rp.Element.(*java.If); ok {
			call = v.matchEquality(ifStmt)
		}
		if call == nil {
			newStmts = append(newStmts, rp)
			continue
		}
		changed = true
		newStmts = append(newStmts, java.RightPadded[java.Statement]{
			Element: call,
			After:   rp.After,
			Markers: rp.Markers,
		})
	}

	if !changed {
		return block
	}

	recipegolang.MaybeAddImport(v, v.importPath, nil, false)
	return block.WithStatements(newStmts)
}

// matchEquality returns the assert/require Equal/NotEqual replacement for an
// `if a != b { <reporter>(...) }` / `if a == b { ... }` guard, or nil.
func (v *equalityVisitor) matchEquality(ifStmt *java.If) *java.MethodInvocation {
	if ifStmt.Condition == nil || ifStmt.ElsePart != nil {
		return nil
	}

	bin, ok := ifStmt.Condition.Tree.Element.(*java.Binary)
	if !ok {
		return nil
	}
	assertion := equalityAssertion(bin.Operator.Element)
	if assertion == "" {
		return nil
	}
	// A nil operand is a presence check (err/pointer), handled by the NoError /
	// Error recipes, not an equality assertion.
	if isNilIdent(bin.Left) || isNilIdent(bin.Right) {
		return nil
	}
	// testify Equal/NotEqual compare by value (reflect.DeepEqual); Go `==`/`!=`
	// on pointers and interfaces compare identity. Converting an identity check
	// would change its meaning, so only fire when both operands are value-safe:
	// a literal or a resolved basic type where `==` and DeepEqual agree.
	if !valueComparable(bin.Left) || !valueComparable(bin.Right) {
		return nil
	}

	body, ok := ifStmt.ThenPart.Element.(*java.Block)
	if !ok || len(body.Statements) != 1 {
		return nil
	}
	call, ok := body.Statements[0].Element.(*java.MethodInvocation)
	if !ok {
		return nil
	}
	recv := reporterReceiver(call, v.isReporter)
	if recv == nil {
		return nil
	}

	expected, actual := orderExpectedActual(bin.Left, bin.Right)
	return finishAssertion(ifStmt.GetPrefix(), v.pkg, v.importPath, recv, assertion, call, []java.Expression{expected, actual}, identSet(bin.Left, bin.Right), nil)
}

// valueComparable reports whether an operand is safe for testify's value
// equality: a literal, or a resolved basic type (string / numeric / bool) where
// Go `==` and reflect.DeepEqual agree. Pointer, interface and struct operands
// return false, since their comparison semantics may differ.
func valueComparable(e java.Expression) bool {
	if _, ok := e.(*java.Literal); ok {
		return true
	}
	t := matcher.TypeOfExpression(e)
	return matcher.IsString(t) || matcher.IsNumeric(t) || matcher.IsBool(t)
}

// equalityAssertion maps a comparison operator to the testify assertion whose
// failure it reproduces: `!=` guards assert equality, `==` guards assert
// inequality. Any other operator yields "".
func equalityAssertion(op java.BinaryOperator) string {
	switch op {
	case java.NotEqual:
		return "Equal"
	case java.Equal:
		return "NotEqual"
	}
	return ""
}

// orderExpectedActual returns the two operands ordered as testify expects
// (expected, actual). It ranks each operand by naming and literalness and puts
// the more expected-looking one first; when neither is distinguishable the
// source order is kept, which is correct and only affects failure-message
// labels.
func orderExpectedActual(a, b java.Expression) (expected, actual java.Expression) {
	if expectedRank(b) > expectedRank(a) {
		return b, a
	}
	return a, b
}

func expectedRank(expr java.Expression) int {
	switch leafName(expr) {
	case "want", "wanted", "expected", "expect", "exp":
		return 2
	case "got", "actual", "result", "res", "have":
		return -2
	}
	if _, ok := expr.(*java.Literal); ok {
		return 1
	}
	return 0
}

// leafName returns the identifier name to rank on: the name for an identifier,
// or the trailing field name for a field access (e.g. `tt.want` -> "want").
func leafName(expr java.Expression) string {
	switch e := expr.(type) {
	case *java.Identifier:
		return e.Name
	case *java.FieldAccess:
		if e.Name.Element != nil {
			return e.Name.Element.Name
		}
	}
	return ""
}

// collectIdentNames adds the value-identifier names referenced in expr to names,
// returning false when expr contains a node shape it does not understand so the
// caller can treat the expression as unsafe to drop.
func collectIdentNames(expr java.Expression, names map[string]bool) bool {
	switch e := expr.(type) {
	case *java.Identifier:
		names[e.Name] = true
		return true
	case *java.Literal:
		return true
	case *java.MethodInvocation:
		if e.Select != nil && !collectIdentNames(e.Select.Element, names) {
			return false
		}
		for _, rp := range e.Arguments.Elements {
			if !collectIdentNames(rp.Element, names) {
				return false
			}
		}
		return true
	case *java.FieldAccess:
		return collectIdentNames(e.Target, names)
	case *java.Binary:
		return collectIdentNames(e.Left, names) && collectIdentNames(e.Right, names)
	case *java.Unary:
		return collectIdentNames(e.Operand, names)
	case *java.Parentheses:
		return collectIdentNames(e.Tree.Element, names)
	case *java.ControlParentheses:
		return collectIdentNames(e.Tree.Element, names)
	}
	return false
}
