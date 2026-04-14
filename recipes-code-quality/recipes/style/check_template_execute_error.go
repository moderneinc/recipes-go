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

func (v *checkTemplateExecuteErrorVisitor) VisitMethodDeclaration(md *tree.MethodDeclaration, p any) tree.J {
	old := v.returnsError
	v.returnsError = funcReturnsError(md)
	result := v.GoVisitor.VisitMethodDeclaration(md, p)
	v.returnsError = old
	return result
}

func (v *checkTemplateExecuteErrorVisitor) VisitBlock(block *tree.Block, p any) tree.J {
	block = v.GoVisitor.VisitBlock(block, p).(*tree.Block)

	changed := false
	var newStmts []tree.RightPadded[tree.Statement]

	for _, rp := range block.Statements {
		mi, ok := rp.Element.(*tree.MethodInvocation)
		if !ok || !isTemplateExecuteCall(mi) {
			newStmts = append(newStmts, rp)
			continue
		}

		if !v.returnsError {
			// Can't auto-wrap: leave a markup hint.
			mi = mi.WithMarkers(tree.MarkupInfo(mi.Markers, "ensure template execute error is checked"))
			newStmts = append(newStmts, tree.RightPadded[tree.Statement]{
				Element: mi, After: rp.After, Markers: rp.Markers,
			})
			continue
		}

		changed = true
		ifStmt := buildIfInitErrCheck(mi)
		newStmts = append(newStmts, tree.RightPadded[tree.Statement]{
			Element: ifStmt, After: rp.After, Markers: rp.Markers,
		})
	}

	if !changed {
		return block
	}
	return block.WithStatements(newStmts)
}

// isTemplateExecuteCall returns true if mi is *.Execute(...) or *.ExecuteTemplate(...).
func isTemplateExecuteCall(mi *tree.MethodInvocation) bool {
	if mi.Select == nil {
		return false
	}
	return mi.Name.Name == "Execute" || mi.Name.Name == "ExecuteTemplate"
}

// funcReturnsError returns true when the last return type of md is the
// identifier "error".
func funcReturnsError(md *tree.MethodDeclaration) bool {
	if md.ReturnType == nil {
		return false
	}
	switch rt := md.ReturnType.(type) {
	case *tree.Identifier:
		return rt.Name == "error"
	case *tree.TypeList:
		types := rt.Types.Elements
		if len(types) == 0 {
			return false
		}
		last := types[len(types)-1].Element
		if vd, ok := last.(*tree.VariableDeclarations); ok {
			if ident, ok2 := vd.TypeExpr.(*tree.Identifier); ok2 {
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
func buildIfInitErrCheck(mi *tree.MethodInvocation) *tree.If {
	// The leading whitespace for a statement-level MethodInvocation typically
	// lives on the Select element (the receiver identifier), not on the
	// MethodInvocation node itself.
	prefix := extractMIPrefix(mi)
	indent := prefix.Indent()

	// Strip the leading whitespace from the call so it fits after `:= `.
	callStripped := stripMIPrefix(mi)

	// err := <call>
	initAssign := &tree.Assignment{
		ID: uuid.New(),
		Variable: &tree.Identifier{
			ID:     uuid.New(),
			Prefix: tree.SingleSpace,
			Name:   "err",
		},
		Markers: tree.Markers{
			ID:      uuid.New(),
			Entries: []tree.Marker{tree.ShortVarDecl{Ident: uuid.New()}},
		},
		Value: tree.LeftPadded[tree.Expression]{
			Before:  tree.SingleSpace,
			Element: callStripped,
		},
	}

	// err != nil
	condition := &tree.Binary{
		ID: uuid.New(),
		Left: &tree.Identifier{
			ID:     uuid.New(),
			Prefix: tree.SingleSpace,
			Name:   "err",
		},
		Operator: tree.LeftPadded[tree.BinaryOperator]{
			Before:  tree.SingleSpace,
			Element: tree.NotEqual,
		},
		Right: &tree.Identifier{
			ID:     uuid.New(),
			Prefix: tree.SingleSpace,
			Name:   "nil",
		},
	}

	// return err
	returnStmt := &tree.Return{
		ID:     uuid.New(),
		Prefix: tree.Space{Whitespace: "\n" + indent + "\t"},
		Expressions: []tree.RightPadded[tree.Expression]{
			{Element: &tree.Identifier{
				ID:     uuid.New(),
				Prefix: tree.SingleSpace,
				Name:   "err",
			}},
		},
	}

	thenBlock := &tree.Block{
		ID:     uuid.New(),
		Prefix: tree.SingleSpace,
		Statements: []tree.RightPadded[tree.Statement]{
			{Element: returnStmt},
		},
		End: tree.Space{Whitespace: "\n" + indent},
	}

	return &tree.If{
		ID:     uuid.New(),
		Prefix: prefix,
		Init: &tree.RightPadded[tree.Statement]{
			Element: initAssign,
		},
		Condition: condition,
		Then:      thenBlock,
	}
}

// extractMIPrefix returns the leading whitespace for a MethodInvocation.
// For `tmpl.Execute(...)`, the indent lives on the Select element (the `tmpl`
// identifier), not on the MethodInvocation node.
func extractMIPrefix(mi *tree.MethodInvocation) tree.Space {
	if mi.Select != nil {
		if ident, ok := mi.Select.Element.(*tree.Identifier); ok && ident.Prefix.Whitespace != "" {
			return ident.Prefix
		}
	}
	return mi.Prefix
}

// stripMIPrefix returns a copy of mi with the leading whitespace replaced by
// a single space so it reads correctly as an RHS expression after `:=`.
func stripMIPrefix(mi *tree.MethodInvocation) *tree.MethodInvocation {
	c := *mi
	if c.Select != nil {
		if ident, ok := c.Select.Element.(*tree.Identifier); ok {
			stripped := ident.WithPrefix(tree.SingleSpace)
			c.Select = &tree.RightPadded[tree.Expression]{
				Element: stripped,
				After:   c.Select.After,
				Markers: c.Select.Markers,
			}
		}
	}
	c.Prefix = tree.EmptySpace
	return &c
}
