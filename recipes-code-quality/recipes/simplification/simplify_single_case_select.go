/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package simplification

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// SimplifySingleCaseSelect replaces `select` statements with exactly one
// communication clause and no default case with the bare channel operation
// and body statements. A single-case select without default is equivalent to
// the bare channel operation.
type SimplifySingleCaseSelect struct {
	recipe.Base
}

func (r *SimplifySingleCaseSelect) Name() string {
	return "org.openrewrite.golang.codequality.SimplifySingleCaseSelect"
}
func (r *SimplifySingleCaseSelect) DisplayName() string { return "Simplify single-case select" }
func (r *SimplifySingleCaseSelect) Description() string {
	return "Replace `select` statements with a single case and no default with the channel operation directly."
}
func (r *SimplifySingleCaseSelect) Tags() []string { return []string{"simplification", "cleanup"} }

func (r *SimplifySingleCaseSelect) Editor() recipe.TreeVisitor {
	return visitor.Init(&findSingleCaseSelectVisitor{})
}

type findSingleCaseSelectVisitor struct {
	visitor.GoVisitor
}

// isSingleCaseSelect returns the single CommClause if sw is a select with
// exactly one communication clause and no default, or nil otherwise.
func isSingleCaseSelect(sw *java.Switch) *golang.CommClause {
	if !java.HasMarker[golang.SelectStmt](sw.Markers) {
		return nil
	}
	if sw.Body == nil {
		return nil
	}
	var theClause *golang.CommClause
	clauses := 0
	for _, stmt := range sw.Body.Statements {
		if cc, ok := stmt.Element.(*golang.CommClause); ok {
			clauses++
			if cc.Comm == nil {
				return nil // has default
			}
			theClause = cc
		}
	}
	if clauses != 1 {
		return nil
	}
	return theClause
}

// VisitBlock replaces single-case select statements with their channel
// operation and body statements, spliced directly into the enclosing block.
func (v *findSingleCaseSelectVisitor) VisitBlock(block *java.Block, p any) java.J {
	block = v.GoVisitor.VisitBlock(block, p).(*java.Block)

	changed := false
	var newStmts []java.RightPadded[java.Statement]
	dedent := visitor.Init(&selectSingleDedentVisitor{})

	for _, rp := range block.Statements {
		sw, ok := rp.Element.(*java.Switch)
		if !ok {
			newStmts = append(newStmts, rp)
			continue
		}

		clause := isSingleCaseSelect(sw)
		if clause == nil {
			newStmts = append(newStmts, rp)
			continue
		}

		changed = true

		// The comm statement (e.g., `v := <-ch`) has a prefix that was the
		// space between "case" and the expression (typically just " ").
		// Replace the entire leading whitespace of the comm with the
		// select statement's prefix (e.g., "\n\t") so it sits at the right
		// indentation level.
		commFixed := replaceLeadingPrefix(clause.Comm, sw.Prefix)
		newStmts = append(newStmts, java.RightPadded[java.Statement]{Element: commFixed})

		// Body statements are indented for the case body (2 levels deeper
		// than the function body). Dedent by 1 tab to match the select's
		// indentation level.
		for _, bodyRP := range clause.Body {
			bodyDedented := dedent.Visit(bodyRP.Element, nil).(java.Statement)
			newStmts = append(newStmts, java.RightPadded[java.Statement]{
				Element: bodyDedented,
				After:   bodyRP.After,
				Markers: bodyRP.Markers,
			})
		}
	}

	if !changed {
		return block
	}

	block = block.WithStatements(newStmts)
	return block
}

// replaceLeadingPrefix replaces all leading whitespace of a statement with
// the given prefix. For compound nodes like Assignment, this means setting
// the node prefix and clearing the first child's prefix.
func replaceLeadingPrefix(stmt java.Statement, prefix java.Space) java.Statement {
	switch s := stmt.(type) {
	case *java.Assignment:
		c := *s
		c.Prefix = prefix
		// Clear the variable's prefix since it was the visual prefix before "case" was stripped.
		switch v := c.Variable.(type) {
		case *java.Identifier:
			c.Variable = v.WithPrefix(java.EmptySpace)
		}
		return &c
	case *golang.Send:
		return s.WithPrefix(prefix)
	case *java.Unary:
		return s.WithPrefix(prefix)
	case *java.MethodInvocation:
		return s.WithPrefix(prefix)
	default:
		return stmt
	}
}

// selectSingleDedentVisitor removes one tab from every whitespace in a subtree
// to adjust for the removed case indentation.
type selectSingleDedentVisitor struct {
	visitor.GoVisitor
}

func (v *selectSingleDedentVisitor) VisitSpace(space java.Space, p any) java.Space {
	if strings.Contains(space.Whitespace, "\t") {
		space.Whitespace = strings.Replace(space.Whitespace, "\t", "", 1)
	}
	return space
}
