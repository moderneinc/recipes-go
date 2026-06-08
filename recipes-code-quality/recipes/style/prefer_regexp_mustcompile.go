/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/google/uuid"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// PreferRegexpMustCompile collapses the boilerplate error-check pattern
//
//	x, err := regexp.Compile(p)
//	if err != nil {
//	    // handle
//	}
//
// into `x := regexp.MustCompile(p)`. MustCompile panics on an invalid pattern,
// which is the desired behaviour for regexes that are expected to always
// compile (the original error branch is removed).
//
// The recipe is deliberately narrow: it only fires when the `regexp.Compile`
// call is the sole RHS of a two-variable short declaration whose second
// variable is the one tested by an immediately-following `if err != nil` guard.
// A blind `Compile` -> `MustCompile` swap is unsafe because `MustCompile`
// returns a single value, so it would turn the two-value assignment into
// non-compiling Go.
type PreferRegexpMustCompile struct {
	recipe.Base
}

func (r *PreferRegexpMustCompile) Name() string {
	return "org.openrewrite.golang.codequality.PreferRegexpMustCompile"
}
func (r *PreferRegexpMustCompile) DisplayName() string {
	return "Prefer regexp.MustCompile for unchecked patterns"
}
func (r *PreferRegexpMustCompile) Description() string {
	return "Collapse `x, err := regexp.Compile(p)` followed by an `if err != nil` guard into `x := regexp.MustCompile(p)`."
}
func (r *PreferRegexpMustCompile) Tags() []string { return []string{"style"} }

func (r *PreferRegexpMustCompile) Editor() recipe.TreeVisitor {
	return visitor.Init(&preferRegexpMustCompileVisitor{})
}

type preferRegexpMustCompileVisitor struct {
	visitor.GoVisitor
}

func (v *preferRegexpMustCompileVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	stmts := block.Statements
	changed := false
	var newStmts []java.RightPadded[java.Statement]

	for i := 0; i < len(stmts); i++ {
		assign, errName, call := matchCompileAssign(stmts[i].Element)
		if assign == nil || i+1 >= len(stmts) || !isErrGuard(stmts[i+1].Element, errName) {
			newStmts = append(newStmts, stmts[i])
			continue
		}

		// Removing the declaration of `errName` is only safe if it is not read
		// after the guard we are about to delete.
		if identUsedIn(stmts[i+2:], errName) {
			newStmts = append(newStmts, stmts[i])
			continue
		}

		newStmts = append(newStmts, java.RightPadded[java.Statement]{
			Element: collapseToMustCompile(assign, call),
			After:   stmts[i].After,
			Markers: stmts[i].Markers,
		})
		i++ // consume the `if err != nil` guard
		changed = true
	}

	if !changed {
		return block
	}
	return block.WithStatements(newStmts)
}

// matchCompileAssign returns the assignment, the name of the error variable and
// the `regexp.Compile` call when stmt is `x, err := regexp.Compile(p)`. It
// returns nils otherwise.
func matchCompileAssign(stmt java.Statement) (*golang.MultiAssignment, string, *java.MethodInvocation) {
	ma, ok := stmt.(*golang.MultiAssignment)
	if !ok || !java.HasMarker[golang.ShortVarDecl](ma.Markers) {
		return nil, "", nil
	}
	if len(ma.Variables) != 2 || len(ma.Values) != 1 {
		return nil, "", nil
	}
	errIdent, ok := ma.Variables[1].Element.(*java.Identifier)
	if !ok || errIdent.Name == "_" {
		return nil, "", nil
	}
	call := regexpCompileCall(ma.Values[0].Element)
	if call == nil {
		return nil, "", nil
	}
	return ma, errIdent.Name, call
}

// regexpCompileCall returns the call expression when expr is `regexp.Compile(...)`.
func regexpCompileCall(expr java.Expression) *java.MethodInvocation {
	mi, ok := expr.(*java.MethodInvocation)
	if !ok || mi.Select == nil || mi.Name.Name != "Compile" {
		return nil
	}
	ident, ok := mi.Select.Element.(*java.Identifier)
	if !ok || ident.Name != "regexp" {
		return nil
	}
	return mi
}

// isErrGuard reports whether stmt is `if <errName> != nil { ... }` with no init
// clause and no else branch.
func isErrGuard(stmt java.Statement, errName string) bool {
	ifStmt, ok := stmt.(*java.If)
	if !ok || ifStmt.Init != nil || ifStmt.ElsePart != nil {
		return false
	}
	bin, ok := ifStmt.Condition.(*java.Binary)
	if !ok || bin.Operator.Element != java.NotEqual {
		return false
	}
	left, lok := bin.Left.(*java.Identifier)
	right, rok := bin.Right.(*java.Identifier)
	return lok && rok && left.Name == errName && right.Name == "nil"
}

// collapseToMustCompile rewrites the two-variable assignment into a single
// `x := regexp.MustCompile(p)` short declaration.
func collapseToMustCompile(ma *golang.MultiAssignment, call *java.MethodInvocation) *java.Assignment {
	mustCall := *call
	mustName := *call.Name
	mustName.Name = "MustCompile"
	mustCall.Name = &mustName

	return &java.Assignment{
		ID:       uuid.New(),
		Prefix:   ma.Prefix,
		Markers:  ma.Markers,
		Variable: ma.Variables[0].Element,
		Value: java.LeftPadded[java.Expression]{
			Before:  ma.Operator.Before,
			Element: &mustCall,
		},
	}
}

// identUsedIn reports whether any statement references an identifier named name.
func identUsedIn(stmts []java.RightPadded[java.Statement], name string) bool {
	c := &identUsageVisitor{name: name}
	c.Self = c
	for _, rp := range stmts {
		c.Visit(rp.Element, nil)
	}
	return c.used
}

type identUsageVisitor struct {
	visitor.GoVisitor
	name string
	used bool
}

func (v *identUsageVisitor) VisitIdentifier(ident *java.Identifier, p any) java.J {
	if ident.Name == v.name {
		v.used = true
	}
	return v.GoVisitor.VisitIdentifier(ident, p)
}
