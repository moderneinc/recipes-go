/*
 * Moderne Proprietary. Only for use by Moderne customers under the terms of a commercial contract.
 */

package redundancy

import (
	"github.com/moderneinc/recipes-go/diagnostic"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/printer"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/recipe"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/golang"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/tree/java"
	"github.com/openrewrite/rewrite/rewrite-go/pkg/visitor"
)

// AllBranchesIdentical replaces an if/else chain whose branches all contain
// identical code with just the body. When every branch does the same thing the
// condition is meaningless and adds unnecessary complexity.
//
//	if a { x() } else { x() }  ->  { x() }
//	if a { x() } else if b { x() } else { x() }  ->  { x() }
//
// The recipe only fires when a final else clause is present (otherwise not all
// paths are covered) and every branch body prints identically.
type AllBranchesIdentical struct {
	recipe.Base
}

func (r *AllBranchesIdentical) Name() string {
	return "org.openrewrite.golang.codequality.AllBranchesIdentical"
}
func (r *AllBranchesIdentical) DisplayName() string {
	return "Remove if/else with identical branches"
}
func (r *AllBranchesIdentical) Description() string {
	return "Replace an if/else chain where every branch has the same body with just the body, " +
		"since the conditions have no effect on the outcome."
}
func (r *AllBranchesIdentical) Tags() []string {
	return []string{"cleanup", "redundancy", "RSPEC-S3923"}
}

func (r *AllBranchesIdentical) DiagnosticMappings() []diagnostic.Mapping {
	return []diagnostic.Mapping{
		{DiagnosticID: "dupBranchBody", Tool: diagnostic.GolangciLint, HasFix: true},
	}
}

func (r *AllBranchesIdentical) Editor() recipe.TreeVisitor {
	return visitor.Init(&allBranchesIdenticalVisitor{})
}

type allBranchesIdenticalVisitor struct {
	visitor.GoVisitor
}

func (v *allBranchesIdenticalVisitor) VisitIf(ifStmt *java.If, p any) java.J {
	ifStmt = v.GoVisitor.VisitIf(ifStmt, p).(*java.If)

	// An `if init; cond {}` runs its init unconditionally and may declare a
	// variable the body relies on, so collapsing the chain would drop that
	// statement and change behavior (or stop compiling).
	if parent := v.Cursor().Parent(); parent != nil {
		if _, wrapped := parent.Value().(*golang.StatementWithInit); wrapped {
			return ifStmt
		}
	}

	if !allBranchBodiesIdentical(ifStmt) {
		return ifStmt
	}

	// Replace the entire if/else chain with just the then-body, preserving
	// the if statement's prefix (indentation).
	thenBlock, ok := ifStmt.ThenPart.Element.(*java.Block)
	if !ok {
		return ifStmt
	}
	return thenBlock.WithPrefix(ifStmt.Prefix)
}

// allBranchBodiesIdentical walks the if/else-if/else chain and returns true
// only when a final else clause exists and every branch body is identical.
func allBranchBodiesIdentical(ifStmt *java.If) bool {
	thenBlock, ok := ifStmt.ThenPart.Element.(*java.Block)
	if !ok {
		return false
	}
	reference := printBlockNormalized(thenBlock)
	current := ifStmt

	for {
		// Collapsing the chain stops evaluating this condition, which is only
		// safe when the condition is pure.
		if conditionMayHaveSideEffects(current.Condition) {
			return false
		}

		if current.ElsePart == nil {
			// No final else -- not all paths are covered.
			return false
		}

		switch elseBody := current.ElsePart.Body.Element.(type) {
		case *java.If:
			innerThen, ok := elseBody.ThenPart.Element.(*java.Block)
			if !ok || printBlockNormalized(innerThen) != reference {
				return false
			}
			current = elseBody
		case *java.Block:
			return printBlockNormalized(elseBody) == reference
		default:
			return false
		}
	}
}

func printBlockNormalized(block *java.Block) string {
	return printer.Print(block.WithPrefix(java.Space{}))
}

// conditionMayHaveSideEffects reports whether evaluating a condition might
// mutate state, perform I/O, or communicate on a channel. In Go an expression
// can only do any of those through a function or method call (every call, incl.
// generic instantiations and calls through function values, is a
// MethodInvocation) or a channel receive, so flagging those two is complete for
// that class -- conversions (TypeCast) and composite literals are pure and are
// correctly left alone. Panic ordering (a nil deref, out-of-range index, failed
// type assertion or integer division by zero in the condition) is not treated
// as a side effect here, matching the upstream rewrite-static-analysis recipe.
func conditionMayHaveSideEffects(cond *java.ControlParentheses) bool {
	if cond == nil {
		return false
	}
	scan := &sideEffectScanner{}
	scan.Self = scan
	scan.Visit(cond.Tree.Element, nil)
	return scan.found
}

type sideEffectScanner struct {
	visitor.GoVisitor
	found bool
}

func (s *sideEffectScanner) VisitMethodInvocation(mi *java.MethodInvocation, p any) java.J {
	s.found = true
	return mi
}

func (s *sideEffectScanner) VisitGoUnary(unary *golang.Unary, p any) java.J {
	if unary.Operator.Element == golang.Receive {
		s.found = true
	}
	return s.GoVisitor.VisitGoUnary(unary, p)
}
