/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"github.com/google/uuid"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// UseCommaOkTypeAssertion transforms bare type assertions `v := x.(T)` into
// the comma-ok form `v, ok := x.(T)` with `_ = ok` to suppress unused-var
// errors. Bare type assertions panic on failure; the comma-ok form is safer.
type UseCommaOkTypeAssertion struct {
	recipe.Base
}

func (r *UseCommaOkTypeAssertion) Name() string {
	return "org.openrewrite.golang.codequality.UseCommaOkTypeAssertion"
}
func (r *UseCommaOkTypeAssertion) DisplayName() string {
	return "Use comma-ok type assertion"
}
func (r *UseCommaOkTypeAssertion) Description() string {
	return "Transform bare type assertions `v := x.(T)` into `v, ok := x.(T)` with `_ = ok` to avoid panics on assertion failure."
}
func (r *UseCommaOkTypeAssertion) Tags() []string { return []string{"style", "lint"} }

func (r *UseCommaOkTypeAssertion) Editor() recipe.TreeVisitor {
	return visitor.Init(&useCommaOkTypeAssertionVisitor{})
}

type useCommaOkTypeAssertionVisitor struct {
	visitor.GoVisitor
}

func (v *useCommaOkTypeAssertionVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	changed := false
	var newStmts []tree.RightPadded[tree.Statement]

	for _, rp := range block.Statements {
		assign, ok := rp.Element.(*tree.Assignment)
		if !ok {
			newStmts = append(newStmts, rp)
			continue
		}

		// Must be a short var decl (:=)
		if !tree.HasMarker[tree.ShortVarDecl](assign.Markers) {
			newStmts = append(newStmts, rp)
			continue
		}

		// RHS must be a TypeCast (type assertion)
		_, isCast := assign.Value.Element.(*tree.TypeCast)
		if !isCast {
			newStmts = append(newStmts, rp)
			continue
		}

		// LHS must be a single identifier (not blank _)
		lhsIdent, ok := assign.Variable.(*tree.Identifier)
		if !ok || lhsIdent.Name == "_" {
			newStmts = append(newStmts, rp)
			continue
		}

		changed = true

		// Build: v, ok := x.(T)
		ma := &tree.MultiAssignment{
			ID:      uuid.New(),
			Prefix:  assign.Prefix,
			Markers: assign.Markers,
			Variables: []tree.RightPadded[tree.Expression]{
				{Element: assign.Variable},
				{Element: &tree.Identifier{
					ID:     uuid.New(),
					Prefix: tree.SingleSpace,
					Name:   "ok",
				}},
			},
			Operator: tree.LeftPadded[tree.Space]{Before: assign.Value.Before, Element: tree.EmptySpace},
			Values: []tree.RightPadded[tree.Expression]{
				{Element: assign.Value.Element},
			},
		}

		// Build: _ = ok (to suppress unused variable)
		// The indent lives on the variable identifier, not on Assignment.Prefix.
		stmtIndent := lhsIdent.Prefix
		suppressOk := &tree.Assignment{
			ID:     uuid.New(),
			Prefix: assign.Prefix,
			Variable: &tree.Identifier{
				ID:     uuid.New(),
				Prefix: stmtIndent,
				Name:   "_",
			},
			Value: tree.LeftPadded[tree.Expression]{
				Before: tree.SingleSpace,
				Element: &tree.Identifier{
					ID:     uuid.New(),
					Prefix: tree.SingleSpace,
					Name:   "ok",
				},
			},
		}

		newStmts = append(newStmts,
			tree.RightPadded[tree.Statement]{Element: ma, After: rp.After},
			tree.RightPadded[tree.Statement]{Element: suppressOk, After: rp.After, Markers: rp.Markers},
		)
	}

	if !changed {
		return block
	}
	return block.WithStatements(newStmts)
}
