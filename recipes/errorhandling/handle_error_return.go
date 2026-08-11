/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package errorhandling

import (
	"github.com/google/uuid"
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/moderneinc/recipes-go/recipes/internal/lstutil"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// Replaces a discarded trailing `_` in a `:=` capture with `err` and adds an
// `if err != nil { return err }` guard, firing only in the top-level block of a
// function that returns a single `error`.
// golangci-lint: errcheck
type HandleErrorReturn struct {
	recipe.Base
}

func (r *HandleErrorReturn) Name() string {
	return "org.openrewrite.golang.codequality.HandleErrorReturn"
}
func (r *HandleErrorReturn) DisplayName() string { return "Handle error return value" }
func (r *HandleErrorReturn) Description() string {
	return "Replace discarded `_` error return values with `err` to capture the error."
}
func (r *HandleErrorReturn) Tags() []string { return []string{"errorhandling", "lint"} }

func (r *HandleErrorReturn) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "errcheck", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *HandleErrorReturn) Editor() recipe.TreeVisitor {
	return visitor.Init(&handleErrorReturnVisitor{})
}

type handleErrorReturnVisitor struct {
	visitor.GoVisitor
}

func (v *handleErrorReturnVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	// Only rewrite the function's top-level block, since a `return err` in a
	// nested loop or if would change control flow.
	if !lstutil.IsFunctionBodyBlock(v.Cursor()) {
		return block
	}

	// Bail unless the enclosing function returns a single error, so `return err`
	// compiles and the captured `err` is used.
	if !enclosingReturnsSingleError(v.Cursor()) {
		return block
	}

	changed := false
	var newStmts []java.RightPadded[java.Statement]
	for _, rp := range block.Statements {
		ma, ok := rp.Element.(*golang.MultiAssignment)
		if !ok || !capturesDiscardedError(ma) || !discardsAnError(ma) {
			newStmts = append(newStmts, rp)
			continue
		}

		newStmts = append(newStmts, java.RightPadded[java.Statement]{
			Element: withErrCapture(ma),
			After:   rp.After,
			Markers: rp.Markers,
		})
		newStmts = append(newStmts, java.RightPadded[java.Statement]{
			Element: buildReturnErrGuard(lstutil.BaseIndent(ma.Prefix)),
		})
		changed = true
	}

	if !changed {
		return block
	}
	return block.WithStatements(newStmts)
}

// Reports whether ma is a `:=` declaration whose last variable is `_` and which
// declares at least one non-blank variable.
func capturesDiscardedError(ma *golang.MultiAssignment) bool {
	if !java.HasMarker[golang.ShortVarDecl](ma.Markers) || len(ma.Variables) < 2 {
		return false
	}
	last, ok := ma.Variables[len(ma.Variables)-1].Element.(*java.Identifier)
	if !ok || last.Name != "_" {
		return false
	}
	for _, v := range ma.Variables[:len(ma.Variables)-1] {
		if id, ok := v.Element.(*java.Identifier); ok && id.Name != "_" {
			return true
		}
	}
	return false
}

// Reports whether ma's value is a function call whose last result is of type
// error, excluding comma-ok forms and non-error last results.
func discardsAnError(ma *golang.MultiAssignment) bool {
	if len(ma.Values) != 1 {
		return false
	}
	mi, ok := ma.Values[0].Element.(*java.MethodInvocation)
	if !ok || mi.MethodType == nil {
		return false
	}
	pz, ok := mi.MethodType.ReturnType.(*java.JavaTypeParameterized)
	if !ok || len(pz.TypeParameters) == 0 {
		return false
	}
	last := pz.TypeParameters[len(pz.TypeParameters)-1]
	return java.TypeSignature(last) == "error"
}

// Returns a copy of ma with the trailing blank identifier renamed to `err`.
func withErrCapture(ma *golang.MultiAssignment) *golang.MultiAssignment {
	lastVar := ma.Variables[len(ma.Variables)-1]
	replaced := lastVar.Element.(*java.Identifier).WithName("err")
	vars := make([]java.RightPadded[java.Expression], len(ma.Variables))
	copy(vars, ma.Variables)
	vars[len(vars)-1] = java.RightPadded[java.Expression]{
		Element: replaced,
		After:   lastVar.After,
		Markers: lastVar.Markers,
	}
	c := *ma
	c.Variables = vars
	return &c
}

// Constructs `if err != nil { return err }`, indented to sit at the same level
// (base) as the assignment it follows.
func buildReturnErrGuard(base string) *java.If {
	cond := &java.ControlParentheses{
		ID: uuid.New(),
		Tree: java.RightPadded[java.Expression]{Element: &java.Binary{
			ID:       uuid.New(),
			Left:     &java.Identifier{ID: uuid.New(), Prefix: java.SingleSpace, Name: "err"},
			Operator: java.LeftPadded[java.BinaryOperator]{Before: java.SingleSpace, Element: java.NotEqual},
			Right:    &java.Identifier{ID: uuid.New(), Prefix: java.SingleSpace, Name: "nil"},
		}},
	}

	ret := &java.Return{
		ID:         uuid.New(),
		Prefix:     java.Space{Whitespace: "\n" + base + "\t"},
		Expression: &java.Identifier{ID: uuid.New(), Prefix: java.SingleSpace, Name: "err"},
	}

	guardBody := &java.Block{
		ID:         uuid.New(),
		Prefix:     java.SingleSpace,
		Statements: []java.RightPadded[java.Statement]{{Element: ret}},
		End:        java.Space{Whitespace: "\n" + base},
	}

	return &java.If{
		ID:        uuid.New(),
		Prefix:    java.Space{Whitespace: "\n" + base},
		Condition: cond,
		ThenPart:  java.RightPadded[java.Statement]{Element: guardBody},
	}
}
