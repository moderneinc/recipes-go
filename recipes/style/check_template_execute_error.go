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

// CheckTemplateExecuteError wraps bare calls to `template.Execute` and
// `template.ExecuteTemplate` in an if-init error check:
//
//	if err := tmpl.Execute(w, data); err != nil { return err }
//
// When the enclosing function does not return error the call is marked
// with MarkupInfo instead.
type CheckTemplateExecuteError struct {
	recipe.Base
}

func (r *CheckTemplateExecuteError) Name() string {
	return "org.openrewrite.golang.codequality.CheckTemplateExecuteError"
}
func (r *CheckTemplateExecuteError) DisplayName() string { return "Check template execute error" }
func (r *CheckTemplateExecuteError) Description() string {
	return "Wrap bare calls to `Execute` and `ExecuteTemplate` on templates in an if-init error check so the returned error is not silently ignored."
}
func (r *CheckTemplateExecuteError) Tags() []string { return []string{"style", "html/template"} }

func (r *CheckTemplateExecuteError) Editor() recipe.TreeVisitor {
	return visitor.Init(&checkTemplateExecuteErrorVisitor{})
}

type checkTemplateExecuteErrorVisitor struct {
	visitor.GoVisitor
	returnsError bool
}

func (v *checkTemplateExecuteErrorVisitor) VisitMethodDeclaration(md *java.MethodDeclaration, p any) java.J {
	old := v.returnsError
	v.returnsError = funcReturnsError(md)
	result := v.GoVisitor.VisitMethodDeclaration(md, p)
	v.returnsError = old
	return result
}

func (v *checkTemplateExecuteErrorVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	changed := false
	var newStmts []java.RightPadded[java.Statement]

	for _, rp := range block.Statements {
		mi, ok := rp.Element.(*java.MethodInvocation)
		if !ok || !isTemplateExecuteCall(mi) {
			newStmts = append(newStmts, rp)
			continue
		}

		if !v.returnsError {
			// Can't auto-wrap: leave a markup hint.
			mi = mi.WithMarkers(java.MarkupInfo(mi.Markers, "ensure template execute error is checked"))
			newStmts = append(newStmts, java.RightPadded[java.Statement]{
				Element: mi, After: rp.After, Markers: rp.Markers,
			})
			continue
		}

		changed = true
		ifStmt := buildIfInitErrCheck(mi)
		newStmts = append(newStmts, java.RightPadded[java.Statement]{
			Element: ifStmt, After: rp.After, Markers: rp.Markers,
		})
	}

	if !changed {
		return block
	}
	return block.WithStatements(newStmts)
}

// isTemplateExecuteCall returns true if mi is *.Execute(...) or *.ExecuteTemplate(...).
func isTemplateExecuteCall(mi *java.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	return mi.Name.Name == "Execute" || mi.Name.Name == "ExecuteTemplate"
}

// Reports whether md returns a single error result, the only case where the
// synthesized `return err` compiles.
func funcReturnsError(md *java.MethodDeclaration) bool {
	if md.ReturnType == nil {
		return false
	}
	switch rt := md.ReturnType.(type) {
	case *java.Identifier:
		return rt.Name == "error"
	case *golang.TypeList:
		types := rt.Types.Elements
		if len(types) != 1 {
			return false
		}
		if vd, ok := types[0].Element.(*java.VariableDeclarations); ok {
			if ident, ok2 := vd.TypeExpr.(*java.Identifier); ok2 {
				return ident.Name == "error"
			}
		}
	}
	return false
}

// buildIfInitErrCheck constructs:
//
//	if err := <call>; err != nil {
//	    return err
//	}
func buildIfInitErrCheck(mi *java.MethodInvocation) *golang.StatementWithInit {
	// The leading whitespace for a statement-level MethodInvocation typically
	// lives on the Select element (the receiver identifier), not on the
	// MethodInvocation node itself.
	prefix := extractMIPrefix(mi)
	indent := prefix.Indent()

	// Strip the leading whitespace from the call so it fits after `:= `.
	callStripped := stripMIPrefix(mi)

	// err := <call>
	initAssign := &java.Assignment{
		ID: uuid.New(),
		Variable: &java.Identifier{
			ID:     uuid.New(),
			Prefix: java.SingleSpace,
			Name:   "err",
		},
		Markers: java.Markers{
			ID:      uuid.New(),
			Entries: []java.Marker{golang.ShortVarDecl{Ident: uuid.New()}},
		},
		Value: java.LeftPadded[java.Expression]{
			Before:  java.SingleSpace,
			Element: callStripped,
		},
	}

	// err != nil
	condition := &java.Binary{
		ID: uuid.New(),
		Left: &java.Identifier{
			ID:     uuid.New(),
			Prefix: java.SingleSpace,
			Name:   "err",
		},
		Operator: java.LeftPadded[java.BinaryOperator]{
			Before:  java.SingleSpace,
			Element: java.NotEqual,
		},
		Right: &java.Identifier{
			ID:     uuid.New(),
			Prefix: java.SingleSpace,
			Name:   "nil",
		},
	}

	// return err
	returnStmt := &java.Return{
		ID:     uuid.New(),
		Prefix: java.Space{Whitespace: "\n" + indent + "\t"},
		Expression: &java.Identifier{
			ID:     uuid.New(),
			Prefix: java.SingleSpace,
			Name:   "err",
		},
	}

	thenBlock := &java.Block{
		ID:     uuid.New(),
		Prefix: java.SingleSpace,
		Statements: []java.RightPadded[java.Statement]{
			{Element: returnStmt},
		},
		End: java.Space{Whitespace: "\n" + indent},
	}

	// An `if init; cond` is modeled as a golang.StatementWithInit wrapping a plain
	// java.If: the wrapper owns the keyword prefix and the init clause, and the
	// inner If's condition is a ControlParentheses (Go elides the parens). The
	// `;` between init and condition comes from a Semicolon marker on the
	// init's padding.
	innerIf := &java.If{
		ID: uuid.New(),
		Condition: &java.ControlParentheses{
			ID:   uuid.New(),
			Tree: java.RightPadded[java.Expression]{Element: condition},
		},
		ThenPart: java.RightPadded[java.Statement]{Element: thenBlock},
	}

	return &golang.StatementWithInit{
		ID:     uuid.New(),
		Prefix: prefix,
		Init: java.RightPadded[java.Statement]{
			Element: initAssign,
			Markers: java.Markers{
				ID:      uuid.New(),
				Entries: []java.Marker{golang.NewSemicolon()},
			},
		},
		Statement: innerIf,
	}
}

// extractMIPrefix returns the leading whitespace for a MethodInvocation.
// For `tmpl.Execute(...)`, the indent lives on the Select element (the `tmpl`
// identifier), not on the MethodInvocation node.
func extractMIPrefix(mi *java.MethodInvocation) java.Space {
	if mi.Select != nil {
		if ident, ok := mi.Select.Element.(*java.Identifier); ok && ident.Prefix.Whitespace != "" {
			return ident.Prefix
		}
	}
	return mi.Prefix
}

// stripMIPrefix returns a copy of mi with the leading whitespace replaced by
// a single space so it reads correctly as an RHS expression after `:=`.
func stripMIPrefix(mi *java.MethodInvocation) *java.MethodInvocation {
	c := *mi
	if c.Select != nil {
		if ident, ok := c.Select.Element.(*java.Identifier); ok {
			stripped := ident.WithPrefix(java.SingleSpace)
			c.Select = &java.RightPadded[java.Expression]{
				Element: stripped,
				After:   c.Select.After,
				Markers: c.Select.Markers,
			}
		}
	}
	c.Prefix = java.EmptySpace
	return &c
}
