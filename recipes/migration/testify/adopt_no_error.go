/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package testify

import (
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	recipegolang "github.com/openrewrite/rewrite/rewrite-go/pkg/recipe/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

const (
	requirePkg    = "require"
	requireImport = "github.com/stretchr/testify/require"
	assertPkg     = "assert"
	assertImport  = "github.com/stretchr/testify/assert"
)

// AdoptTestifyRequireNoError replaces `if err != nil { t.Fatal(err) }` guards in
// tests with `require.NoError(t, err)`. The fatal reporter (`t.Fatal`/`t.Fatalf`/
// `t.FailNow`) maps the guard to `require`, since both abort the test on failure.
type AdoptTestifyRequireNoError struct {
	recipe.Base
}

func (r *AdoptTestifyRequireNoError) Name() string {
	return "org.openrewrite.golang.testify.AdoptTestifyRequireNoError"
}
func (r *AdoptTestifyRequireNoError) DisplayName() string {
	return "Adopt testify require.NoError"
}
func (r *AdoptTestifyRequireNoError) Description() string {
	return "Replace `if err != nil { t.Fatal(err) }` guards in tests with `require.NoError(t, err)` from `github.com/stretchr/testify/require`."
}
func (r *AdoptTestifyRequireNoError) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestifyRequireNoError) Editor() recipe.TreeVisitor {
	return visitor.Init(&errGuardVisitor{pkg: requirePkg, importPath: requireImport, isReporter: isFatalReporter, op: java.NotEqual, assertion: "NoError"})
}

// AdoptTestifyAssertNoError replaces `if err != nil { t.Error(err) }` guards in
// tests with `assert.NoError(t, err)`. The non-fatal reporter (`t.Error`/
// `t.Errorf`/`t.Fail`) maps the guard to `assert`, which reports but keeps going.
type AdoptTestifyAssertNoError struct {
	recipe.Base
}

func (r *AdoptTestifyAssertNoError) Name() string {
	return "org.openrewrite.golang.testify.AdoptTestifyAssertNoError"
}
func (r *AdoptTestifyAssertNoError) DisplayName() string {
	return "Adopt testify assert.NoError"
}
func (r *AdoptTestifyAssertNoError) Description() string {
	return "Replace `if err != nil { t.Error(err) }` guards in tests with `assert.NoError(t, err)` from `github.com/stretchr/testify/assert`."
}
func (r *AdoptTestifyAssertNoError) Tags() []string { return []string{"testing", "testify"} }

func (r *AdoptTestifyAssertNoError) Editor() recipe.TreeVisitor {
	return visitor.Init(&errGuardVisitor{pkg: assertPkg, importPath: assertImport, isReporter: isErrorReporter, op: java.NotEqual, assertion: "NoError"})
}

// errGuardVisitor rewrites `if err <op> nil { <reporter>(...) }` guards to
// `<pkg>.<assertion>(t, err)`. The fields select the flavour: op is NotEqual for
// NoError and Equal for Error; pkg / importPath / isReporter select require vs
// assert.
type errGuardVisitor struct {
	visitor.GoVisitor
	pkg        string
	importPath string
	isReporter func(string) bool
	op         java.BinaryOperator
	assertion  string
}

func (v *errGuardVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	changed := false
	newStmts := make([]java.RightPadded[java.Statement], 0, len(block.Statements))
	for _, rp := range block.Statements {
		var call *java.MethodInvocation
		switch el := rp.Element.(type) {
		case *java.If:
			call = v.matchNilGuard(el)
		case *golang.StatementWithInit:
			call = v.matchInlineInitGuard(el)
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

// matchNilGuard returns the `<pkg>.NoError(t, err)` replacement for an
// `if err != nil { <reporter>(err) }` statement, or nil when it is not one.
func (v *errGuardVisitor) matchNilGuard(ifStmt *java.If) *java.MethodInvocation {
	if ifStmt.Condition == nil || ifStmt.ElsePart != nil {
		return nil
	}

	bin, ok := ifStmt.Condition.Tree.Element.(*java.Binary)
	if !ok || bin.Operator.Element != v.op {
		return nil
	}
	errExpr := errNonNilOperand(bin)
	if errExpr == nil {
		return nil
	}

	call, recv := reporterBodyCall(ifStmt.ThenPart.Element, v.isReporter)
	if call == nil {
		return nil
	}
	return finishAssertion(ifStmt.GetPrefix(), v.pkg, v.importPath, recv, v.assertion, call, []java.Expression{errExpr}, identSet(errExpr), nil)
}

// matchInlineInitGuard returns the replacement for the inline-init form
// `if err := f(); err != nil { <reporter>(err) }`, inlining the call to
// `<pkg>.NoError(t, f())`. Restricted to a single `:=`-declared variable so
// nothing else can reference it, which keeps the inline safe.
func (v *errGuardVisitor) matchInlineInitGuard(swi *golang.StatementWithInit) *java.MethodInvocation {
	ifStmt, ok := swi.Statement.(*java.If)
	if !ok || ifStmt.Condition == nil || ifStmt.ElsePart != nil {
		return nil
	}

	varIdent, valueExpr := singleShortVarDecl(swi.Init.Element)
	if varIdent == nil || !looksLikeError(varIdent) {
		return nil
	}

	bin, ok := ifStmt.Condition.Tree.Element.(*java.Binary)
	if !ok || bin.Operator.Element != v.op || !comparesIdentToNil(bin, varIdent.Name) {
		return nil
	}

	call, recv := reporterBodyCall(ifStmt.ThenPart.Element, v.isReporter)
	if call == nil {
		return nil
	}
	// The init variable is consumed by inlining, so a message that references it
	// must not survive (it would be undefined); mark it a survivor so its dumps
	// strip, and forbidden so any surviving reference drops the message.
	surv := identSet(valueExpr)
	surv[varIdent.Name] = true
	forbidden := map[string]bool{varIdent.Name: true}
	return finishAssertion(swi.GetPrefix(), v.pkg, v.importPath, recv, v.assertion, call, []java.Expression{valueExpr}, surv, forbidden)
}

func isFatalReporter(name string) bool {
	switch name {
	case "Fatal", "Fatalf", "FailNow":
		return true
	}
	return false
}

func isErrorReporter(name string) bool {
	switch name {
	case "Error", "Errorf", "Fail":
		return true
	}
	return false
}

// reporterReceiver returns the receiver identifier of a reporter call
// (per isReporter) on a testing.T-like value, or nil.
func reporterReceiver(call *java.MethodInvocation, isReporter func(string) bool) *java.Identifier {
	if call.Select == nil || call.Name == nil {
		return nil
	}
	recv, ok := call.Select.Element.(*java.Identifier)
	if !ok || !isTestingReceiver(recv) || !isReporter(call.Name.Name) {
		return nil
	}
	return recv
}

// singleShortVarDecl returns the variable and value of a single-variable `:=`
// declaration, or (nil, nil) for anything else. A one-variable `:=` is a
// *java.Assignment; the multi-variable form is a *golang.MultiAssignment.
func singleShortVarDecl(stmt java.Statement) (*java.Identifier, java.Expression) {
	switch a := stmt.(type) {
	case *java.Assignment:
		if !java.HasMarker[golang.ShortVarDecl](a.Markers) {
			return nil, nil
		}
		id, ok := a.Variable.(*java.Identifier)
		if !ok {
			return nil, nil
		}
		return id, a.Value.Element
	case *golang.MultiAssignment:
		if !java.HasMarker[golang.ShortVarDecl](a.Markers) || len(a.Variables) != 1 || len(a.Values) != 1 {
			return nil, nil
		}
		id, ok := a.Variables[0].Element.(*java.Identifier)
		if !ok {
			return nil, nil
		}
		return id, a.Values[0].Element
	}
	return nil, nil
}

// comparesIdentToNil reports whether bin compares the identifier named name
// against nil (in either operand order).
func comparesIdentToNil(bin *java.Binary, name string) bool {
	if isNilIdent(bin.Right) {
		if id, ok := bin.Left.(*java.Identifier); ok && id.Name == name {
			return true
		}
	}
	if isNilIdent(bin.Left) {
		if id, ok := bin.Right.(*java.Identifier); ok && id.Name == name {
			return true
		}
	}
	return false
}

// errNonNilOperand returns the error-typed operand of an `X != nil` comparison,
// or nil when neither side is nil or the other side is not an error.
func errNonNilOperand(bin *java.Binary) java.Expression {
	if isNilIdent(bin.Right) && looksLikeError(bin.Left) {
		return bin.Left
	}
	if isNilIdent(bin.Left) && looksLikeError(bin.Right) {
		return bin.Right
	}
	return nil
}

func isNilIdent(expr java.Expression) bool {
	id, ok := expr.(*java.Identifier)
	return ok && id.Name == "nil"
}

// looksLikeError decides from the resolved type when present, falling back to the
// `err` naming convention when the type is unresolved.
func looksLikeError(expr java.Expression) bool {
	if t := matcher.TypeOfExpression(expr); t != nil {
		return matcher.IsAssignableTo(t, "error")
	}
	id, ok := expr.(*java.Identifier)
	return ok && id.Name == "err"
}

// isTestingReceiver reports whether recv is a testing.T/B/F/TB value, deciding
// from the resolved type when present and otherwise from the common receiver
// naming convention. The resolved-type check excludes package-qualified calls
// such as `log.Fatal`.
func isTestingReceiver(recv *java.Identifier) bool {
	if recv.Type != nil {
		switch java.TypeSignature(recv.Type) {
		case "testing.T", "testing.B", "testing.F", "testing.TB":
			return true
		}
		return false
	}
	switch recv.Name {
	case "t", "b", "tb":
		return true
	}
	return false
}
