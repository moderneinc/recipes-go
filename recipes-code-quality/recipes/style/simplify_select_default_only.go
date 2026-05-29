/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package style

import (
	"strings"

	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// SimplifySelectDefaultOnly replaces `select { default: ... }` statements where
// the select has only a default case and no communication cases with the body
// statements directly. The select wrapper is unnecessary.
type SimplifySelectDefaultOnly struct {
	recipe.Base
}

func (r *SimplifySelectDefaultOnly) Name() string {
	return "org.openrewrite.golang.codequality.SimplifySelectDefaultOnly"
}
func (r *SimplifySelectDefaultOnly) DisplayName() string { return "Simplify select default only" }
func (r *SimplifySelectDefaultOnly) Description() string {
	return "Replace `select { default: ... }` with the body statements when the select has only a default case and no communication cases."
}
func (r *SimplifySelectDefaultOnly) Tags() []string { return []string{"style"} }

func (r *SimplifySelectDefaultOnly) Editor() recipe.TreeVisitor {
	return visitor.Init(&simplifySelectDefaultOnlyVisitor{})
}

type simplifySelectDefaultOnlyVisitor struct {
	visitor.GoVisitor
}

func (v *simplifySelectDefaultOnlyVisitor) VisitSwitch(sw *java.Switch, p any) java.J {
	sw = v.GoVisitor.VisitSwitch(sw, p).(*java.Switch)

	// Only select statements (Switch with SelectStmt marker)
	if !java.HasMarker[golang.SelectStmt](sw.Markers) {
		return sw
	}

	if sw.Body == nil {
		return sw
	}

	// Find the single default CommClause.
	var defaultClause *golang.CommClause
	clauses := 0
	for _, stmt := range sw.Body.Statements {
		cc, ok := stmt.Element.(*golang.CommClause)
		if !ok {
			continue
		}
		clauses++
		if cc.Comm != nil {
			return sw // has a real communication case; leave as-is
		}
		defaultClause = cc
	}

	if clauses != 1 || defaultClause == nil {
		return sw
	}

	// Extract the body statements from the default clause.
	if len(defaultClause.Body) == 0 {
		return &java.Empty{Prefix: sw.Prefix}
	}

	// Dedent the body statements since they are being lifted out of the select block.
	dedent := visitor.Init(&selectDedentVisitor{})

	// For a single statement, return it directly with the select's prefix.
	if len(defaultClause.Body) == 1 {
		stmt := defaultClause.Body[0].Element
		result := dedent.Visit(stmt, p)
		return result.(java.Statement)
	}

	// For multiple statements, return a Block without braces containing the body.
	stmts := make([]java.RightPadded[java.Statement], len(defaultClause.Body))
	for i, rp := range defaultClause.Body {
		dedented := dedent.Visit(rp.Element, p).(java.Statement)
		stmts[i] = java.RightPadded[java.Statement]{
			Element: dedented,
			After:   rp.After,
			Markers: rp.Markers,
		}
	}
	return &java.Block{
		Prefix:     sw.Prefix,
		Statements: stmts,
	}
}

// selectDedentVisitor removes one tab from every whitespace in a subtree.
type selectDedentVisitor struct {
	visitor.GoVisitor
}

func (v *selectDedentVisitor) VisitSpace(space java.Space, p any) java.Space {
	if strings.Contains(space.Whitespace, "\t") {
		space.Whitespace = strings.Replace(space.Whitespace, "\t", "", 1)
	}
	return space
}
