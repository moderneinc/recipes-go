/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/matcher"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
)

// acquisition is an assignment whose value hands the caller a resource that has
// to be released again.
type acquisition struct {
	varName string
	varType java.JavaType
	call    *java.MethodInvocation
	errName string // the error assigned alongside the resource, "" if there is none
}

// extractAcquisition reads an assignment of the form `x := call()` or
// `x, err := call()`.
func extractAcquisition(stmt java.Statement) (acquisition, bool) {
	switch s := stmt.(type) {
	case *java.Assignment:
		call, ok := s.Value.Element.(*java.MethodInvocation)
		if !ok {
			return acquisition{}, false
		}
		ident, ok := s.Variable.(*java.Identifier)
		if !ok {
			return acquisition{}, false
		}
		return acquisition{varName: ident.Name, varType: ident.Type, call: call}, true
	case *golang.MultiAssignment:
		if len(s.Values) == 0 || len(s.Variables) == 0 {
			return acquisition{}, false
		}
		call, ok := s.Values[0].Element.(*java.MethodInvocation)
		if !ok {
			return acquisition{}, false
		}
		ident, ok := s.Variables[0].Element.(*java.Identifier)
		if !ok {
			return acquisition{}, false
		}
		return acquisition{varName: ident.Name, varType: ident.Type, call: call, errName: errVarName(s)}, true
	}
	return acquisition{}, false
}

// errVarName returns the name of the error-typed variable in a multi-value
// assignment, preferring the declared type and falling back to the conventional
// name when the assignment did not resolve.
func errVarName(s *golang.MultiAssignment) string {
	for _, rp := range s.Variables[1:] {
		ident, ok := rp.Element.(*java.Identifier)
		if !ok {
			continue
		}
		if matcher.IsError(ident.Type) || ident.Name == "err" {
			return ident.Name
		}
	}
	return ""
}

// typeIs reports whether t is the named type, seeing through the pointer a Go
// constructor usually returns. Recipes in this family qualify an acquisition by
// the type acquired, because `Query`, `Do`, `Prepare` and `Begin` are all method
// names that unrelated APIs share, and a cleanup call inserted for one of those
// does not compile.
func typeIs(t java.JavaType, fqn string) bool {
	return t != nil && matcher.GetFullyQualifiedName(t) == fqn
}

// declaringType names the type whose method mi invokes. It reports false for an
// unresolved receiver, which matcher.DeclaringTypeFQN would otherwise report as
// the receiver's own identifier — letting a local variable named `os` or `http`
// pass for the package of the same name.
func declaringType(mi *java.MethodInvocation) (string, bool) {
	if mi == nil || mi.MethodType == nil || mi.MethodType.DeclaringType == nil {
		return "", false
	}
	return mi.MethodType.DeclaringType.GetFullyQualifiedName(), true
}

// deferIndex gives the position in stmts at which the cleanup for the
// acquisition at index i belongs: after the `if err != nil` block guarding it,
// since the resource is only valid once that check has passed. Only defers may
// sit between the acquisition and its guard.
func deferIndex(stmts []java.RightPadded[java.Statement], i int, errName string) int {
	if errName == "" {
		return i + 1
	}
	for j := i + 1; j < len(stmts); j++ {
		if _, isDefer := stmts[j].Element.(*golang.Defer); isDefer {
			continue
		}
		ifStmt, ok := stmts[j].Element.(*java.If)
		if !ok || ifStmt.Condition == nil {
			break
		}
		if !lstutil.IsNotNilCheck(ifStmt.Condition.Tree.Element, errName) {
			break
		}
		return j + 1
	}
	return i + 1
}

// insertDefer adds a cleanup statement for every acquisition in the block that
// match accepts and present has not already covered.
func insertDefer(block *java.Block,
	match func(acquisition) bool,
	present func(stmts []java.RightPadded[java.Statement], from int, varName string) bool,
	build func(a acquisition, acquire java.Statement) *golang.Defer,
) *java.Block {
	pending := map[int][]java.RightPadded[java.Statement]{}
	for i, rp := range block.Statements {
		a, ok := extractAcquisition(rp.Element)
		if !ok || !match(a) || present(block.Statements, i, a.varName) {
			continue
		}
		at := deferIndex(block.Statements, i, a.errName)
		pending[at] = append(pending[at], java.RightPadded[java.Statement]{Element: build(a, rp.Element)})
	}
	if len(pending) == 0 {
		return block
	}

	var stmts []java.RightPadded[java.Statement]
	for i, rp := range block.Statements {
		stmts = append(stmts, rp)
		stmts = append(stmts, pending[i+1]...)
	}
	return block.WithStatements(stmts)
}

// insertDeferMethodCall covers the common cleanup shape, `defer x.Close()`.
func insertDeferMethodCall(block *java.Block, match func(acquisition) bool, method string) *java.Block {
	return insertDefer(block, match,
		func(stmts []java.RightPadded[java.Statement], from int, varName string) bool {
			return hasDeferAfter(stmts, from, varName, method)
		},
		func(a acquisition, acquire java.Statement) *golang.Defer {
			return buildDeferMethodCall(a, method, acquire)
		})
}

// hasDeferAfter checks whether any statement after index i in the block is a
// defer calling varName.methodName().
func hasDeferAfter(stmts []java.RightPadded[java.Statement], i int, varName, methodName string) bool {
	for j := i + 1; j < len(stmts); j++ {
		d, ok := stmts[j].Element.(*golang.Defer)
		if !ok {
			continue
		}
		if matchesDeferCall(d, varName, methodName) {
			return true
		}
	}
	return false
}

// matchesDeferCall returns true if the defer calls varName.methodName().
func matchesDeferCall(d *golang.Defer, varName, methodName string) bool {
	mi, ok := d.Expr.(*java.MethodInvocation)
	if !ok {
		return false
	}
	if mi.Name.Name != methodName {
		return false
	}
	if mi.Select == nil {
		return false
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok {
		return false
	}
	return ident.Name == varName
}

// buildDeferMethodCall builds a `defer varName.methodName()` statement,
// copying the indentation prefix from the original statement. The receiver's
// resolved type supplies both the qualifier's type and the method's.
func buildDeferMethodCall(a acquisition, methodName string, originalStmt java.Statement) *golang.Defer {
	selectIdent := &java.Identifier{
		ID:   uuid.New(),
		Name: a.varName,
		Type: a.varType,
	}
	methodIdent := &java.Identifier{
		ID:   uuid.New(),
		Name: methodName,
	}
	closeCall := &java.MethodInvocation{
		ID:     uuid.New(),
		Prefix: java.SingleSpace,
		Select: &java.RightPadded[java.Expression]{Element: selectIdent},
		Name:   methodIdent,
		Arguments: java.Container[java.Expression]{
			Before: java.EmptySpace,
		},
		MethodType: lstutil.MethodOn(a.varType, methodName),
	}
	return &golang.Defer{
		ID:     uuid.New(),
		Prefix: stmtPrefix(originalStmt),
		Expr:   closeCall,
	}
}

// stmtPrefix extracts the leading whitespace from a statement.
// In Go's AST the indentation lives on the first token of the statement,
// not on the statement node's own Prefix field.
func stmtPrefix(stmt java.Statement) java.Space {
	switch s := stmt.(type) {
	case *java.Assignment:
		if id, ok := s.Variable.(*java.Identifier); ok && id.Prefix.Whitespace != "" {
			return id.Prefix
		}
		return s.Prefix
	case *golang.MultiAssignment:
		if len(s.Variables) > 0 {
			if id, ok := s.Variables[0].Element.(*java.Identifier); ok && id.Prefix.Whitespace != "" {
				return id.Prefix
			}
		}
		return s.Prefix
	case *golang.Defer:
		return s.Prefix
	case *java.MethodInvocation:
		return s.Prefix
	default:
		return java.Space{Whitespace: "\n\t"}
	}
}
